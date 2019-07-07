package interop

import (
	"testing"
)

func TestIterators(t *testing.T) {
	expect(t, `
	out = {};
	for k, v in go_map {
		out[string(k)] = v
	}
	`, MAP{
		"go_map": map[string]int{"a": 1, "b": 2},
	}, MAP{"a": 1, "b": 2})

	expect(t, `
	out = {};
	for k, v in go_map {
		out[string(k)] = v
	}
	`, MAP{
		"go_map": map[int]string{1: "a", 2: "b"},
	}, MAP{"1": "a", "2": "b"})

	expect(t, `
	out = [];
	for i, v in go_slice {
		out = append(out, format("%d:%v", i, v))
	}
	`, MAP{
		"go_slice": []int16{1, 2, 3, 4, 5},
	}, ARR{"0:1", "1:2", "2:3", "3:4", "4:5"})

	goChan := make(chan int)
	go func() {
		goChan <- 1
		goChan <- 2
		goChan <- 3
		goChan <- 4
		close(goChan)
	}()
	expect(t, `
	out = [];
	for i, v in go_chan {
		out = append(out, format("%d:%v", i, v))
	}
	`, MAP{
		"go_chan": goChan,
	}, ARR{"0:1", "1:2", "2:3", "3:4"})
}
