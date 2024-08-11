package character

import (
	"log/slog"
	"time"

	"github.com/hectorgimenez/d2go/pkg/data/state"
	"github.com/hectorgimenez/koolo/internal/game"

	"github.com/hectorgimenez/d2go/pkg/data"
	"github.com/hectorgimenez/d2go/pkg/data/npc"
	"github.com/hectorgimenez/d2go/pkg/data/skill"
	"github.com/hectorgimenez/d2go/pkg/data/stat"
	"github.com/hectorgimenez/koolo/internal/action"
	"github.com/hectorgimenez/koolo/internal/action/step"
	"github.com/hectorgimenez/koolo/internal/pather"
)

const (
	sorceressMaxAttacksLoop = 40
	sorceressMinDistance    = 25
	sorceressMaxDistance    = 30
	blizzardCooldown        = time.Second * 4
)

type BlizzardSorceress struct {
	BaseCharacter
}

func (s BlizzardSorceress) CheckKeyBindings(d game.Data) []skill.ID {
	requiredKeybindings := []skill.ID{skill.Blizzard, skill.Teleport, skill.TomeOfTownPortal, skill.ShiverArmor, skill.StaticField}
	missingKeybindings := []skill.ID{}

	for _, cskill := range requiredKeybindings {
		if _, found := d.KeyBindings.KeyBindingForSkill(cskill); !found {
			if cskill == skill.ShiverArmor {
				if !s.hasAnyArmorKeyBinding(d) {
					missingKeybindings = append(missingKeybindings, skill.ShiverArmor)
				}
			} else {
				missingKeybindings = append(missingKeybindings, cskill)
			}
		}
	}

	if len(missingKeybindings) > 0 {
		s.logger.Debug("Missing required key bindings", slog.Any("Bindings", missingKeybindings))
	}

	return missingKeybindings
}

func (s BlizzardSorceress) hasAnyArmorKeyBinding(d game.Data) bool {
	armors := []skill.ID{skill.FrozenArmor, skill.ChillingArmor, skill.ShiverArmor}
	for _, armor := range armors {
		if _, found := d.KeyBindings.KeyBindingForSkill(armor); found {
			return true
		}
	}
	return false
}

func (s BlizzardSorceress) KillMonsterSequence(
	monsterSelector func(d game.Data) (data.UnitID, bool),
	skipOnImmunities []stat.Resist,
	opts ...step.AttackOption,
) action.Action {
	completedAttackLoops := 0
	previousUnitID := 0
	previousSelfBlizzard := time.Time{}

	return action.NewStepChain(func(d game.Data) []step.Step {
		id, found := monsterSelector(d)
		if !found {
			return nil
		}
		if previousUnitID != int(id) {
			completedAttackLoops = 0
		}

		if !s.preBattleChecks(d, id, skipOnImmunities) {
			return nil
		}

		if len(opts) == 0 {
			opts = append(opts, step.Distance(sorceressMinDistance, sorceressMaxDistance))
		}

		if completedAttackLoops >= sorceressMaxAttacksLoop {
			return nil
		}

		// Cast a Blizzard on very close mobs to clear nearby trash, every two attack rotations
		if time.Since(previousSelfBlizzard) > blizzardCooldown && !d.PlayerUnit.States.HasState(state.Cooldown) {
			for _, m := range d.Monsters.Enemies() {
				if dist := pather.DistanceFromMe(d, m.Position); dist < 4 {
					s.logger.Debug("Casting Blizzard on nearby monster", slog.Any("Monster", m.Name))
					previousSelfBlizzard = time.Now()
					return []step.Step{step.SecondaryAttack(skill.Blizzard, m.UnitID, 1, opts...)}
				}
			}
		}

		// Reduce attack distance if monster is unreachable
		if completedAttackLoops > 12 {
			if completedAttackLoops == 13 {
				s.logger.Debug("Reducing attack distance due to unreachable monster")
			}
			opts = []step.AttackOption{step.Distance(1, 5)}
		}

		completedAttackLoops++
		previousUnitID = int(id)

		if d.PlayerUnit.States.HasState(state.Cooldown) {
			return []step.Step{step.PrimaryAttack(id, 2, true, opts...)}
		}

		return []step.Step{step.SecondaryAttack(skill.Blizzard, id, 1, opts...)}
	}, action.RepeatUntilNoSteps())
}

func (s BlizzardSorceress) BuffSkills(d game.Data) []skill.ID {
	var skillsList []skill.ID
	if _, found := d.KeyBindings.KeyBindingForSkill(skill.EnergyShield); found {
		skillsList = append(skillsList, skill.EnergyShield)
	}

	if armor := s.findActiveArmorSkill(d); armor != skill.IDNone {
		skillsList = append(skillsList, armor)
	}

	return skillsList
}

func (s BlizzardSorceress) findActiveArmorSkill(d game.Data) skill.ID {
	armors := []skill.ID{skill.ChillingArmor, skill.ShiverArmor, skill.FrozenArmor}
	for _, armor := range armors {
		if _, found := d.KeyBindings.KeyBindingForSkill(armor); found {
			return armor
		}
	}
	return skill.IDNone
}

func (s BlizzardSorceress) PreCTABuffSkills(d game.Data) []skill.ID {
	return nil
}

func (s BlizzardSorceress) KillMonsterByName(id npc.ID, monsterType data.MonsterType, maxDistance int, skipOnImmunities []stat.Resist) action.Action {
	return s.KillMonsterSequence(func(d game.Data) (data.UnitID, bool) {
		if m, found := d.Monsters.FindOne(id, monsterType); found {
			return m.UnitID, true
		}
		return 0, false
	}, skipOnImmunities, step.Distance(sorceressMinDistance, maxDistance))
}

func (s BlizzardSorceress) KillCountess() action.Action {
	return s.KillMonsterByName(npc.DarkStalker, data.MonsterTypeSuperUnique, sorceressMaxDistance, nil)
}

func (s BlizzardSorceress) KillAndariel() action.Action {
	return s.KillMonsterByName(npc.Andariel, data.MonsterTypeNone, sorceressMaxDistance, nil)
}

func (s BlizzardSorceress) KillSummoner() action.Action {
	return s.KillMonsterByName(npc.Summoner, data.MonsterTypeNone, sorceressMaxDistance, nil)
}

func (s BlizzardSorceress) KillDuriel() action.Action {
	return s.KillMonsterByName(npc.Duriel, data.MonsterTypeNone, sorceressMaxDistance, nil)
}

func (s BlizzardSorceress) KillPindle(skipOnImmunities []stat.Resist) action.Action {
	return s.KillMonsterByName(npc.DefiledWarrior, data.MonsterTypeSuperUnique, sorceressMaxDistance, skipOnImmunities)
}

func (s BlizzardSorceress) KillMephisto() action.Action {
	return s.KillMonsterByName(npc.Mephisto, data.MonsterTypeNone, sorceressMaxDistance, nil)
}

func (s BlizzardSorceress) KillNihlathak() action.Action {
	return s.KillMonsterByName(npc.Nihlathak, data.MonsterTypeSuperUnique, sorceressMaxDistance, nil)
}

func (s BlizzardSorceress) KillDiablo() action.Action {
	timeout := time.Second * 20
	startTime := time.Now()
	diabloFound := false

	return action.NewChain(func(d game.Data) []action.Action {
		if time.Since(startTime) > timeout && !diabloFound {
			s.logger.Error("Diablo not found, timeout reached")
			return nil
		}

		diablo, found := d.Monsters.FindOne(npc.Diablo, data.MonsterTypeNone)
		if !found || diablo.Stats[stat.Life] <= 0 {
			if diabloFound {
				return nil
			}

			return []action.Action{action.NewStepChain(func(d game.Data) []step.Step {
				return []step.Step{step.Wait(time.Millisecond * 100)}
			})}
		}

		diabloFound = true
		s.logger.Info("Diablo detected, attacking")

		return []action.Action{
			action.NewStepChain(func(d game.Data) []step.Step {
				return []step.Step{
					step.SecondaryAttack(skill.StaticField, diablo.UnitID, 5, step.Distance(3, 8)),
				}
			}),
			s.killMonster(npc.Diablo, data.MonsterTypeNone),
		}
	}, action.RepeatUntilNoSteps())
}

func (s BlizzardSorceress) KillIzual() action.Action {
	return s.KillRepeatedly(npc.Izual, 7)
}

func (s BlizzardSorceress) KillBaal() action.Action {
	return s.KillRepeatedly(npc.BaalCrab, 5)
}

func (s BlizzardSorceress) KillRepeatedly(npcID npc.ID, staticFieldDistance int) action.Action {
	return action.NewChain(func(d game.Data) []action.Action {
		return []action.Action{
			action.NewStepChain(func(d game.Data) []step.Step {
				m, _ := d.Monsters.FindOne(npcID, data.MonsterTypeNone)
				return []step.Step{
					step.SecondaryAttack(skill.StaticField, m.UnitID, staticFieldDistance, step.Distance(5, 8)),
				}
			}),
			s.killMonster(npcID, data.MonsterTypeNone),
			s.killMonster(npcID, data.MonsterTypeNone),
			s.killMonster(npcID, data.MonsterTypeNone),
			s.killMonster(npcID, data.MonsterTypeNone),
		}
	})
}

func (s BlizzardSorceress) KillCouncil() action.Action {
	return s.KillMonsterSequence(func(d game.Data) (data.UnitID, bool) {
		var councilMembers []data.Monster
		var coldImmunes []data.Monster
		for _, m := range d.Monsters.Enemies() {
			if m.Name == npc.CouncilMember || m.Name == npc.CouncilMember2 || m.Name == npc.CouncilMember3 {
				if m.IsImmune(stat.ColdImmune) {
					coldImmunes = append(coldImmunes, m)
				} else {
					councilMembers = append(councilMembers, m)
				}
			}
		}

		councilMembers = append(councilMembers, coldImmunes...)

		for _, m := range councilMembers {
			return m.UnitID, true
		}

		return 0, false
	}, nil, step.Distance(8, sorceressMaxDistance))
}

func (s BlizzardSorceress) killMonsterByName(id npc.ID, monsterType data.MonsterType, maxDistance int, skipOnImmunities []stat.Resist) action.Action {
	return s.KillMonsterSequence(func(d game.Data) (data.UnitID, bool) {
		if m, found := d.Monsters.FindOne(id, monsterType); found {
			return m.UnitID, true
		}
		return 0, false
	}, skipOnImmunities, step.Distance(sorceressMinDistance, maxDistance))
}

func (s BlizzardSorceress) killMonster(npc npc.ID, t data.MonsterType) action.Action {
	return s.KillMonsterSequence(func(d game.Data) (data.UnitID, bool) {
		m, found := d.Monsters.FindOne(npc, t)
		if !found {
			return 0, false
		}

		return m.UnitID, true
	}, nil)
}
