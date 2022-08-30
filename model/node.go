package model

type Node struct {
	FilePath string
	Path     string
	Name     string
	pMap     map[string]*Node
}
