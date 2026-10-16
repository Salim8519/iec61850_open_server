package sales

import "testing"

func TestAppendKeepsEveryRecordInOrder(t *testing.T) {
	dir := t.TempDir()
	book := Open(dir)
	if records, err := Read(dir); err != nil || len(records) != 0 {
		t.Fatalf("empty book: %v %v", records, err)
	}
	for _, event := range []string{"paid", "refunded"} {
		if err := book.Append(Record{Time: 1, Event: event, OrderID: "ORDER-1", Amount: "149.00"}); err != nil {
			t.Fatal(err)
		}
	}
	records, err := Read(dir)
	if err != nil || len(records) != 2 || records[0].Event != "paid" || records[1].Event != "refunded" {
		t.Fatalf("records: %+v %v", records, err)
	}
}
