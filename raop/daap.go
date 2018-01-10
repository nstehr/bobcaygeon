package raop

import (
	"encoding/binary"
	"log"
)

type contentType struct {
	code  string
	name  string
	cType string
}

// based on: https://github.com/kylewelsby/daap/blob/master/index.js
func parseDaap(daap []byte) {
	i := 8
	parsedData := make(map[string]interface{})
	for i < len(daap) {
		itemType := string(daap[i : i+4])
		itemLength := int(binary.BigEndian.Uint32(daap[i+4 : i+8]))
		if itemLength != 0 {
			data := daap[i+8 : i+8+itemLength]
			contentType := getContentType(itemType)
			switch contentType.cType {
			case "byte":
				parsedData[contentType.name] = data[0]
			case "string":
				parsedData[contentType.name] = string(data)
			}
		}
		i = i + itemLength + 8
	}
	log.Println(parsedData)
}

func getContentType(code string) contentType {
	ct := contentType{}
	// there is a whole TON of types that can come back
	// only parse out the ones we are interested in for now
	switch code {
	case "mikd":
		ct.cType = "byte"
		ct.code = "mikd"
		ct.name = "dmap.itemkind"
	case "asal":
		ct.cType = "string"
		ct.code = "asal"
		ct.name = "daap.songalbum"
	case "asar":
		ct.cType = "string"
		ct.code = "asar"
		ct.name = "daap.songartist"
	case "minm":
		ct.cType = "string"
		ct.code = "mimn"
		ct.name = "dmap.itemname"
	}
	return ct
}
