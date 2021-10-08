package demoinfocs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_calculateFrameRates(t *testing.T) {
	type args struct {
		tickDiffs    map[int]int
		tickInterval float32
	}
	tests := []struct {
		name          string
		args          args
		wantFrameRate float64
	}{
		{
			name:          "simple 128",
			wantFrameRate: 128,
			args: args{
				tickDiffs: map[int]int{
					1: 10,
				},
				tickInterval: 1.0 / 128.0,
			},
		},
		{
			name:          "ignore 0 128",
			wantFrameRate: 128,
			args: args{
				tickDiffs: map[int]int{
					0: 100,
					1: 10,
				},
				tickInterval: 1.0 / 128.0,
			},
		},
		{
			name:          "ignore high 128",
			wantFrameRate: 128,
			args: args{
				tickDiffs: map[int]int{
					17: 100,
					1:  10,
				},
				tickInterval: 1.0 / 128.0,
			},
		},
		{
			name:          "calc 96, pow2 128",
			wantFrameRate: 128,
			args: args{
				tickDiffs: map[int]int{
					17: 100,
					2:  5,
					1:  10,
				},
				tickInterval: 1.0 / 128.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFrameRate := calculateFrameRateBestGuess(tt.args.tickDiffs, tt.args.tickInterval)

			assert.Equal(t, tt.wantFrameRate, gotFrameRate)
		})
	}
}
