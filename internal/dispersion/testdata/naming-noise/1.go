package sample

type Record struct {
	ID   int
	Name string
}

func GetRecord(id int) (Record, error) { return Record{}, nil }

const MaxRecords = 100