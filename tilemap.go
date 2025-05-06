package main

type TileMap struct {
	TileSet struct {
		FirstGID string `xml:"firstgid,attr"`
		Source   string `xml:"source,attr"`
	} `xml:"tileset"`
	Data string `xml:"layer>data"`
}
