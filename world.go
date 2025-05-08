package main

type Map struct {
	Filename string `json:"fileName"`
	Height   int    `json:"Height"`
	Width    int    `json:"Width"`
	WorldX   int    `json:"x"`
	WorldY   int    `json:"y"`
}

type World struct {
	Maps []Map
}
