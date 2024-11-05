package service

// 投票結果の得票数での順位付けができるようsort.Interfaceを実装した
// 参考：https://code-database.com/knowledges/98
type Candidate struct {
	Name  string
	Count int
}

type ByRank []Candidate

func (arr ByRank) Len() int {
	return len(arr)
}

// 降順にソート
func (arr ByRank) Less(i, j int) bool {
	return arr[i].Count > arr[j].Count
}

func (arr ByRank) Swap(i, j int) {
	arr[i], arr[j] = arr[j], arr[i]
}
