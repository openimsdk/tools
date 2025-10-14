package window

// Sum computes the total sum of values within the window
func Sum(iterator Iterator) float64 {
	var sum float64
	for iterator.Next() {
		bucket := iterator.Bucket()
		for _, p := range bucket.Points {
			sum += p
		}
	}
	return sum
}

// Avg computes the average of values within the window
func Avg(iterator Iterator) float64 {
	var sum float64
	var count int
	for iterator.Next() {
		bucket := iterator.Bucket()
		for _, p := range bucket.Points {
			sum += p
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// Max computes the maximum value within the window
func Max(iterator Iterator) float64 {
	var max float64
	var initialized bool
	for iterator.Next() {
		bucket := iterator.Bucket()
		for _, p := range bucket.Points {
			if !initialized {
				max = p
				initialized = true
				continue
			}
			if p > max {
				max = p
			}
		}
	}
	return max
}

// Min computes the minimum value within the window
func Min(iterator Iterator) float64 {
	var min float64
	var initialized bool
	for iterator.Next() {
		bucket := iterator.Bucket()
		for _, p := range bucket.Points {
			if !initialized {
				min = p
				initialized = true
				continue
			}
			if p < min {
				min = p
			}
		}
	}
	return min
}

// Count computes the total number of data points within the window
func Count(iterator Iterator) float64 {
	var count int64
	for iterator.Next() {
		bucket := iterator.Bucket()
		count += bucket.Count
	}
	return float64(count)
}
