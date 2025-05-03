package main

type Map struct {
	Filename string `json:"fileName"`
	Height   int    `json:"Height"`
	Width    int    `json:"Width"`
	XPos     int    `json:"x"`
	YPos     int    `json:"y"`
}

type World struct {
	Maps []Map
}
