package main

type TileProperty struct {
	Name  string `xml:"name,attr"`
	Type  string `xml:"type,attr"`
	Value string `xml:"value,attr"`
}

type Tile struct {
	ID         int            `xml:"id,attr"`
	Properties []TileProperty `xml:"properties>property"`
}

type TileSet struct {
	Tiles []Tile `xml:"tile"`
}
