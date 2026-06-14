package skills

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

var namePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type frontmatter struct {
	Name                   string `yaml:"name"`
	Description            string `yaml:"description"`
	DisableModelInvocation bool   `yaml:"disable-model-invocation"`
}

type Meta struct {
	Name                   string
	Description            string
	DisableModelInvocation bool
}

// Parse แยก YAML frontmatter และ body จากเนื้อหา SKILL.md
func Parse(content []byte) (Meta, string, error) {
	text := string(content)
	text = strings.TrimPrefix(text, "\ufeff")
	if !strings.HasPrefix(text, "---") {
		return Meta{}, "", fmt.Errorf("SKILL.md ต้องขึ้นต้นด้วย YAML frontmatter (---)")
	}

	rest := text[3:]
	if rest != "" && rest[0] == '\n' {
		rest = rest[1:]
	} else if rest != "" && rest[0] == '\r' {
		if len(rest) > 1 && rest[1] == '\n' {
			rest = rest[2:]
		} else {
			rest = rest[1:]
		}
	}

	end := strings.Index(rest, "\n---")
	if end < 0 {
		return Meta{}, "", fmt.Errorf("SKILL.md frontmatter ไม่ปิดด้วย ---")
	}

	fmRaw := rest[:end]
	body := strings.TrimPrefix(rest[end+4:], "\n")
	body = strings.TrimPrefix(body, "\r\n")

	var fm frontmatter
	if err := yaml.Unmarshal([]byte(fmRaw), &fm); err != nil {
		return Meta{}, "", fmt.Errorf("parse frontmatter: %w", err)
	}

	meta := Meta{
		Name:                   strings.TrimSpace(fm.Name),
		Description:            strings.TrimSpace(fm.Description),
		DisableModelInvocation: fm.DisableModelInvocation,
	}
	if err := validateMeta(meta); err != nil {
		return Meta{}, "", err
	}
	return meta, body, nil
}

func validateMeta(meta Meta) error {
	if meta.Name == "" {
		return fmt.Errorf("frontmatter ต้องมี name")
	}
	if utf8.RuneCountInString(meta.Name) > 64 {
		return fmt.Errorf("name ยาวเกิน 64 ตัวอักษร")
	}
	if !namePattern.MatchString(meta.Name) {
		return fmt.Errorf("name ต้องเป็น lowercase letters, numbers, hyphens เท่านั้น")
	}
	if meta.Description == "" {
		return fmt.Errorf("frontmatter ต้องมี description")
	}
	if utf8.RuneCountInString(meta.Description) > 1024 {
		return fmt.Errorf("description ยาวเกิน 1024 ตัวอักษร")
	}
	return nil
}

// ParseFile อ่านและ parse SKILL.md จาก path ที่กำหนด
func ParseFile(path string, data []byte, source Source, fallbackName string) (Skill, error) {
	meta, _, err := Parse(data)
	if err != nil {
		return Skill{}, err
	}
	name := meta.Name
	if name == "" {
		name = fallbackName
	}
	return Skill{
		Name:                   name,
		Description:            meta.Description,
		Path:                   path,
		Source:                 source,
		DisableModelInvocation: meta.DisableModelInvocation,
	}, nil
}
