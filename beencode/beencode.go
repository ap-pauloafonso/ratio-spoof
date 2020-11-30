package beencode

import (
	"strconv"
)

const (
	dictToken                  = byte('d')
	numberToken                = byte('i')
	listToken                  = byte('l')
	endOfCollectionToken       = byte('e')
	lengthValueStringSeparator = byte(':')
)

//Decode accepts a byte slice and returns a map with information parsed.(panic if it fails)
func Decode(data []byte) map[string]interface{} {
	result, _ := findParse(0, &data)
	return result.(map[string]interface{})
}

func findParse(currentIdx int, data *[]byte) (result interface{}, nextIdx int) {
	token := (*data)[currentIdx : currentIdx+1][0]
	switch {
	case token == dictToken:
		return mapParse(currentIdx, data)
	case token == numberToken:
		return numberParse(currentIdx, data)
	case token == listToken:
		return listParse(currentIdx, data)
	case token >= byte('0') || token <= byte('9'):
		return stringParse(currentIdx, data)
	default:
		panic("Error decoding beencode")
	}
}

func mapParse(startIdx int, data *[]byte) (result map[string]interface{}, nextIdx int) {
	result = make(map[string]interface{})
	initialMapIndex := startIdx
	current := startIdx + 1
	for (*data)[current : current+1][0] != endOfCollectionToken {
		mapKey, next := findParse(current, data)
		current = next
		mapValue, next := findParse(current, data)
		current = next
		result[mapKey.(string)] = mapValue
	}
	current++
	result["byte_offsets"] = []int{initialMapIndex, current}
	nextIdx = current
	return
}

func listParse(startIdx int, data *[]byte) (result []interface{}, nextIdx int) {
	current := startIdx + 1
	for (*data)[current : current+1][0] != endOfCollectionToken {
		value, next := findParse(current, data)
		result = append(result, value)
		current = next
	}
	current++
	nextIdx = current
	return
}

func numberParse(startIdx int, data *[]byte) (result int, nextIdx int) {
	current := startIdx
	for (*data)[current : current+1][0] != endOfCollectionToken {
		current++
	}
	value, _ := strconv.Atoi(string((*data)[startIdx+1 : current]))
	result = value
	nextIdx = current + 1
	return
}

func stringParse(startIdx int, data *[]byte) (result string, nextIdx int) {
	current := startIdx
	for (*data)[current : current+1][0] != lengthValueStringSeparator {
		current++
	}
	sizeStr, _ := strconv.Atoi(string(((*data)[startIdx:current])))
	result = string((*data)[current+1 : current+1+int(sizeStr)])
	nextIdx = current + 1 + int(sizeStr)
	return
}
