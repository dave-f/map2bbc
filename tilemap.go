package main

import (
	"errors"
	"strings"
)

type MapProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type MapTileSet struct {
	FirstGID string `xml:"firstgid,attr"`
	Source   string `xml:"source,attr"`
}

type MapLayer struct {
	ID     string `xml:"id,attr"`
	Name   string `xml:"name,attr"`
	Width  int    `xml:"width,attr"`
	Height int    `xml:"height,attr"`
	Locked int    `xml:"locked,attr"`
	Data   string `xml:"data"`
}

type TileMap struct {
	TileSets   []MapTileSet  `xml:"tileset"`
	Properties []MapProperty `xml:"properties>property"`
	Layers     []MapLayer    `xml:"layer"`
}

func (m TileMap) GetMapBaseLayer() (string, error) {

	for _, v := range m.Layers {

		if strings.HasPrefix(v.Name, "Tile Layer") {

			return v.Data, nil
		}
	}

	return "", errors.New("no tile layer")
}
