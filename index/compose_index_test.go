package index

import (
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

// Element 定义元素结构体
type Element struct {
	id int
	A  int
	B  int
	D  int
	F  int
	G  int
	H  int
}

// 构建倒排索引
func buildInvertedIndex(data []Element) (map[int][]Element, map[int][]Element, map[int][]Element) {
	indexA := make(map[int][]Element)
	indexB := make(map[int][]Element)
	indexF := make(map[int][]Element)

	for _, elem := range data {
		indexA[elem.A] = append(indexA[elem.A], elem)
		indexB[elem.B] = append(indexB[elem.B], elem)
		indexF[elem.F] = append(indexF[elem.F], elem)
	}

	// 确保每个倒排索引的列表是有序的
	for key := range indexA {
		sort.Slice(indexA[key], func(i, j int) bool {
			return compareElements(indexA[key][i], indexA[key][j]) < 0
		})
	}
	for key := range indexB {
		sort.Slice(indexB[key], func(i, j int) bool {
			return compareElements(indexB[key][i], indexB[key][j]) < 0
		})
	}
	for key := range indexF {
		sort.Slice(indexF[key], func(i, j int) bool {
			return compareElements(indexF[key][i], indexF[key][j]) < 0
		})
	}

	return indexA, indexB, indexF
}

// 优化交集计算：排序 + 二分查找
func optimizedIntersection(indices ...[]Element) []Element {
	// 按索引长度排序
	sort.Slice(indices, func(i, j int) bool {
		return len(indices[i]) < len(indices[j])
	})

	// 从最小集合开始计算交集
	result := indices[0]
	for i := 1; i < len(indices); i++ {
		result = intersectSorted(result, indices[i])
	}
	return result
}

// 两个有序集合的交集计算
func intersectSorted(a, b []Element) []Element {
	var result []Element
	for _, elem := range a {
		if binarySearch(b, elem) {
			result = append(result, elem)
		}
	}
	return result
}

// 二分查找
func binarySearch(elements []Element, target Element) bool {
	low, high := 0, len(elements)-1
	for low <= high {
		mid := low + (high-low)/2
		if compareElements(elements[mid], target) == 0 {
			return true
		} else if compareElements(elements[mid], target) < 0 {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}
	return false
}

// 比较两个元素
func compareElements(e1, e2 Element) int {
	if e1.A != e2.A {
		return e1.A - e2.A
	}
	if e1.B != e2.B {
		return e1.B - e2.B
	}
	if e1.F != e2.F {
		return e1.F - e2.F
	}
	return e1.id - e2.id
}

func TestGptIndex(t *testing.T) {
	data := []Element{
		{A: 1, B: 3, D: 10, F: 5, G: 100, H: 200},
		{A: 1, B: 3, D: 20, F: 5, G: 101, H: 201},
		{A: 2, B: 4, D: 30, F: 6, G: 102, H: 202},
	}

	// 构建倒排索引
	indexA, indexB, indexF := buildInvertedIndex(data)

	tests := []struct {
		name     string
		a, b, f  int
		expected []Element
	}{
		{"1,3,5", 1, 3, 5, []Element{{A: 1, B: 3, D: 10, F: 5, G: 100, H: 200}, {A: 1, B: 3, D: 20, F: 5, G: 101, H: 201}}},
		{"2,4,6", 2, 4, 6, []Element{{A: 2, B: 4, D: 30, F: 6, G: 102, H: 202}}},
		{"1,3,6", 1, 3, 6, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := optimizedIntersection(indexA[tt.a], indexB[tt.b], indexF[tt.f])
			assert.Equal(t, tt.expected, results)
		})
	}
}
