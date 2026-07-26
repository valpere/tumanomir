package sample

type Record struct {
	Name  string
	ID    int
}

func GetRecord(id int) (Record, error) { return Record{}, nil }

const MaxRecords = 100