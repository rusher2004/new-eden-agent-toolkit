package esi_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestAllPages_ConcatenatesInOrder(t *testing.T) {
	calls := 0
	got, err := esi.AllPages(context.Background(), func(_ context.Context, page int32) ([]int32, int32, error) {
		calls++
		return []int32{page * 10, page*10 + 1}, 3, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []int32{10, 11, 20, 21, 30, 31}
	if calls != 3 || len(got) != len(want) {
		t.Fatalf("got %v after %d calls", got, calls)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestAllPages_WrapsErrorWithPage(t *testing.T) {
	boom := errors.New("boom")
	_, err := esi.AllPages(context.Background(), func(_ context.Context, page int32) ([]int, int32, error) {
		if page == 2 {
			return nil, 0, boom
		}
		return []int{1}, 2, nil
	})
	if !errors.Is(err, boom) || err.Error() != "page 2 of 2: boom" {
		t.Errorf("err = %v", err)
	}
}
