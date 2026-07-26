package sample

type Record struct {
	Identifier  int
	DisplayName string
}

func FetchRecord(id int) (Record, error) { return Record{}, nil }

const RecordLimit = 100