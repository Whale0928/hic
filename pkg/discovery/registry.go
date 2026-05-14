package discovery

type Board struct {
	Agency    string
	BoardKind string
	Name      string
	BaseURL   string
	ListPath  string
	ViewPath  string
}

type StaticSiteRegistry struct {
	boards map[string]Board
}

func NewStaticSiteRegistry() StaticSiteRegistry {
	boards := []Board{
		{
			Agency:    "SH",
			BoardKind: "rental",
			Name:      "SH 주택임대",
			BaseURL:   "https://www.i-sh.co.kr",
			ListPath:  "/app/lay2/program/S48T1581C563/www/brd/m_247/list.do",
			ViewPath:  "/app/lay2/program/S48T1581C563/www/brd/m_247/view.do",
		},
	}

	registry := StaticSiteRegistry{boards: make(map[string]Board, len(boards))}
	for _, board := range boards {
		registry.boards[boardKey(board.Agency, board.BoardKind)] = board
	}
	return registry
}

func (r StaticSiteRegistry) Get(agency string, boardKind string) (Board, bool) {
	board, ok := r.boards[boardKey(agency, boardKind)]
	return board, ok
}

func (r StaticSiteRegistry) List() []Board {
	boards := make([]Board, 0, len(r.boards))
	for _, board := range r.boards {
		boards = append(boards, board)
	}
	return boards
}

func boardKey(agency string, boardKind string) string {
	return agency + ":" + boardKind
}
