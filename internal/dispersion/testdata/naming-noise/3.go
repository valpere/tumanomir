package sample

type Record struct {
	Key   int
	Label string
}

func LookupRecord(id int) (Record, error) { return Record{}, nil }

const MaxEntries = 100