package tmp

type item struct {
	typeIndexValue []int
}

func (i *item) GetIndex(index int) int {
	if len(i.typeIndexValue) > index {
		return i.typeIndexValue[index]
	}
	return 0
}

type indexPos struct {
	group int
	pos   int
}

type indexRange struct {
	indexType  int
	indexValue int

	begin indexPos
	end   indexPos
}

func (p indexRange) Contains(other indexRange) bool {
	return p.begin.pos <= other.begin.pos && p.end.pos >= other.end.pos
}

func (p indexRange) Before(other indexRange) bool {
	return p.end.pos <= other.begin.pos
}

func (p indexRange) After(other indexRange) bool {
	return p.begin.pos >= other.end.pos
}

func (p indexRange) Intersect(other indexRange) (indexRange, bool) {
	// p的起点在other的后面，p的终点也在other的后面
	if p.begin.pos >= other.begin.pos {
		return indexRange{
			indexType:  p.indexType,
			indexValue: p.indexValue,
			begin:      p.begin,
			end:        other.end,
		}, true
	} else {
		// p的起点在other的前面，p的终点也在other的前面
		return indexRange{
			indexType:  p.indexType,
			indexValue: p.indexValue,
			begin:      other.begin,
			end:        p.end,
		}, false
	}
}

var indexCache []*item
var indexTypes [][]indexRange

/*
   items     => sort
1: 1,1,3     =>	1: 113
2: 1,3,4     => 2: 134
3: 2,4,5     => 6: 135
4: 3,3,1     => 5: 221
5: 2,2,2     => 3: 245
6: 1,3,5	 => 4: 331
*/

/*
1: 1:0-3,2:3-5,3:5-6
2: 1:0-1,3:1-3,2:3-4,4:4-5,3:5-6
3: 3:0-1,4:1-2,5:2-3,1:3-4,5:4-5,1:5-6
*/

type nodeRange struct {
	value      int
	begin, end int
	children   []*nodeRange
}

var nodes []*nodeRange

func indexRangeIntersect(target, src []indexRange) {
	targetIndex := 0
	var result []indexRange
	for srcIndex := 0; srcIndex < len(src); srcIndex++ {
		srcRange := src[srcIndex]
		// 可用二分查找
		for ; targetIndex < len(target); targetIndex++ {
			tarRange := target[targetIndex]
			if srcRange.After(tarRange) {
				continue
			}
			if srcRange.Before(target[targetIndex]) {
				break
			}
			/*
			   src:--------
			   dst:    *******

			*/
			if srcRange.Contains(tarRange) {
				result = append(result, target[targetIndex])
			} else {
				intersect, behind := srcRange.Intersect(tarRange)
				result = append(result, intersect)
				if !behind {
					break
				}
			}
		}
	}
}

func findItems(typeIndexValue []int) {
	var resultIndexes []indexRange
	for i, v := range typeIndexValue {
		if v == 0 {
			if i == 0 {
				resultIndexes = append(resultIndexes, indexTypes[0]...)
				continue
			}
			//resultIndexes = append()
			continue
		}
	}
}

func addItem(i *item) {
	//weight := 0
	//var group []*item
	//if len(i.typeIndexValue) <= 2 {
	//	if len(indexCache) == 0 {
	//		indexCache = append(indexCache, make([]*item, 0, 1))
	//	}
	//	group = indexCache[0]
	//}
	//indexCache[0] = append(indexCache[0], i)
}
