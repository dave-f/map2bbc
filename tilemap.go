package main

type MapProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type TileMap struct {
	TileSet struct {
		FirstGID string `xml:"firstgid,attr"`
		Source   string `xml:"source,attr"`
	} `xml:"tileset"`
	Properties []MapProperty `xml:"properties>property"`
	Data       string        `xml:"layer>data"`
}
