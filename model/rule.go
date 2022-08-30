package model

import "html/template"

type typeRule struct {
	Anchor     string `json:"anchor"`
	anchorTmpl *template.Template
	File       string `json:"file"`
	fileTmpl   *template.Template
	tmpl       *template.Template
}

type rules struct {
	Mermaid *typeRule `json:"mermaid"`
}

type Rule struct {
	Constants map[string]string `json:"constants"`
	Rules     rules             `json:"rules"`
}
