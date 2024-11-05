package valueObject

type Page struct {
	Number        int `json:"number"`         // 現在の表示ページ
	Size          int `json:"size"`           // 1ページあたりの件数
	TotalElements int `json:"total_elements"` // 全件数
	TotalPages    int `json:"total_pages"`    // 全ページ
}
