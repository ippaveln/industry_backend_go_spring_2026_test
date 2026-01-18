package main

type Stats struct {
	Count int
	Sum   int64
	Min   int64
	Max   int64
}

func Calc(nums []int64) Stats {
	if len(nums) == 0 {
		return Stats{}
	}

	stats := Stats{
		Count: len(nums),
		Sum:   0,
		Min:   nums[0],
		Max:   nums[0],
	}

	for _, num := range nums {
		stats.Sum += num
		if num < stats.Min {
			stats.Min = num
		}
		if num > stats.Max {
			stats.Max = num
		}
	}

	return stats
}
