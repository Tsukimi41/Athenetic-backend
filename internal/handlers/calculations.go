package handlers

import "math"

// calculateNextReps applies the progression rules to determine the next target.
func calculateNextReps(pastRPE float64, repsCompleted int, targetReps int) int {
	if targetReps <= 0 {
		targetReps = repsCompleted
	}

	rpeRounded := int(math.Round(pastRPE))
	if rpeRounded <= 6 && repsCompleted == targetReps {
		return targetReps + 2
	}
	if rpeRounded >= 7 && rpeRounded <= 8 && repsCompleted == targetReps {
		return targetReps + 1
	}
	if rpeRounded >= 9 || repsCompleted < targetReps {
		return targetReps
	}
	return targetReps + 1
}
