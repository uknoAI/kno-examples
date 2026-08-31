package fixture

import "testing"

const tape = `Output demo.gif
Type 'kno demo' Enter
Sleep 2s
Type '{"id":"refund-01","input":"How do I get a refund?","expected":"Refunds are issued within 5 business days.","tags":["refunds"]}' Enter
Type '{"id":"refund-policy-v3","content":"Refunds are issued within 5 business days.","kind":"knowledge"}' Enter
Sleep 1s
`

// A tape is mostly not fixture data. Only the Type lines carrying a JSON object
// are, and the two streams are told apart by shape rather than by position, so
// a tape that reorders or grows is still compared correctly.
func TestFromTapeSplitsByShapeNotPosition(t *testing.T) {
	cases, pool, err := FromTape([]byte(tape))
	if err != nil {
		t.Fatalf("FromTape: %v", err)
	}
	if len(cases) != 1 || cases[0].ID != "refund-01" {
		t.Errorf("cases = %+v, want just refund-01", cases)
	}
	if len(pool) != 1 || pool[0].ID != "refund-policy-v3" {
		t.Errorf("pool = %+v, want just refund-policy-v3", pool)
	}
}

func TestFromTapeIgnoresEverythingThatIsNotAFixture(t *testing.T) {
	cases, pool, err := FromTape([]byte("Type 'kno demo' Enter\nSleep 2s\nEnter\n"))
	if err != nil {
		t.Fatalf("FromTape: %v", err)
	}
	if len(cases) != 0 || len(pool) != 0 {
		t.Errorf("a tape with no fixture lines yielded %d cases and %d pool assets", len(cases), len(pool))
	}
}

func TestRecordsPreservesExactBytes(t *testing.T) {
	// Two spellings of the same object. Re-marshalling would call them equal;
	// a reader copying one file and a CI job running the other would not.
	a, err := Records([]byte(`{"id":"x","expected":"one"}`))
	if err != nil {
		t.Fatalf("Records: %v", err)
	}
	b, err := Records([]byte(`{"expected":"one","id":"x"}`))
	if err != nil {
		t.Fatalf("Records: %v", err)
	}
	if f := Compare("cases", "a", a, "b", b); len(f) == 0 {
		t.Error("key-order difference was not reported; Records is not preserving raw bytes")
	}
}

func TestCompareReportsBothDirectionsOfAbsence(t *testing.T) {
	a, _ := Records([]byte(`{"id":"only-a","expected":"x"}`))
	b, _ := Records([]byte(`{"id":"only-b","expected":"x"}`))
	f := Compare("cases", "left", a, "right", b)
	if len(f) != 2 {
		t.Fatalf("want a finding for each side, got %d: %v", len(f), f)
	}
}

func TestRecordWithoutAnIDIsAnError(t *testing.T) {
	if _, err := Records([]byte(`{"expected":"x"}`)); err == nil {
		t.Error("a record with no id should be an error: it cannot be matched against another copy")
	}
}

func TestAmbiguousRecordIsAnError(t *testing.T) {
	if _, _, err := FromTape([]byte(`Type '{"id":"x","expected":"a","content":"b"}' Enter`)); err == nil {
		t.Error("a record that is both a Case and a Pool asset should be an error, not silently a Case")
	}
}
