// Package esi's skills.go covers all 3 of "Skills"'s endpoints -- character attributes, skill
// queue, and trained skills. All 3 are private/"my own character" -- verified against
// esi/api/openapi.yaml's security blocks (esi-skills.read_skills.v1 for attributes and skills,
// esi-skills.read_skillqueue.v1 for the queue), never a caller-supplied character ID.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterAttributes returns the client's own character's neural attributes (Charisma,
// Intelligence, Memory, Perception, Willpower) and remap history.
func (c *Client) CharacterAttributes(ctx context.Context) (*esiapi.CharactersCharacterIdAttributesGet, error) {
	result, resp, err := c.api.SkillsAPI.GetCharactersCharacterIdAttributes(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character attributes: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterSkillQueue returns the client's own character's current skill training queue. The
// generated return type is CharactersSkillqueueSkill -- verified directly from source, NOT the
// similarly-named CharactersCharacterIdSkillqueueGetInner (a distinct type used elsewhere in the
// generated client).
func (c *Client) CharacterSkillQueue(ctx context.Context) ([]esiapi.CharactersSkillqueueSkill, error) {
	result, resp, err := c.api.SkillsAPI.GetCharactersCharacterIdSkillqueue(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character skill queue: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterSkills returns the client's own character's trained skills, total skill points, and
// any unallocated skill points.
func (c *Client) CharacterSkills(ctx context.Context) (*esiapi.CharactersSkills, error) {
	result, resp, err := c.api.SkillsAPI.GetCharactersCharacterIdSkills(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character skills: %w", describeError(err, resp))
	}
	return result, nil
}
