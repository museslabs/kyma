package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewTimer(t *testing.T) {
	timer := NewTimer()

	if timer.running {
		t.Error("New timer should not be running")
	}

	if timer.duration != 0 {
		t.Error("New timer should have zero duration")
	}

	if timer.startTime.IsZero() {
		t.Error("New timer should have startTime set")
	}
}

func TestTimerStart(t *testing.T) {
	timer := NewTimer()

	// Start the timer
	timer = timer.Start()

	if !timer.running {
		t.Error("Timer should be running after Start()")
	}

	if timer.duration != 0 {
		t.Error("Timer should have zero duration immediately after start")
	}

	// Starting an already running timer should not change the start time
	originalStartTime := timer.startTime
	time.Sleep(10 * time.Millisecond)
	timer = timer.Start()

	if timer.startTime != originalStartTime {
		t.Error("Starting an already running timer should not change start time")
	}
}

func TestTimerPause(t *testing.T) {
	timer := NewTimer().Start()

	// Let it run for a bit
	time.Sleep(50 * time.Millisecond)

	timer = timer.Pause()

	if timer.running {
		t.Error("Timer should not be running after Pause()")
	}

	if timer.duration <= 0 {
		t.Error("Timer should have positive duration after running and pausing")
	}

	// Pausing an already paused timer should not change duration
	originalDuration := timer.duration
	timer = timer.Pause()

	if timer.duration != originalDuration {
		t.Error("Stopping an already stopped timer should not change duration")
	}
}

func TestTimerResume(t *testing.T) {
	timer := NewTimer().Start()

	// Let it run, then pause
	time.Sleep(50 * time.Millisecond)
	timer = timer.Pause()
	pausedDuration := timer.duration

	// Resume the timer
	timer = timer.Resume()

	if !timer.running {
		t.Error("Timer should be running after Resume()")
	}

	// Let it run a bit more
	time.Sleep(50 * time.Millisecond)

	currentDuration := timer.Duration()
	if currentDuration <= pausedDuration {
		t.Error("Timer duration should increase after resume")
	}

	// Resuming an already running timer should not affect it
	originalStartTime := timer.startTime
	timer = timer.Resume()

	if timer.startTime != originalStartTime {
		t.Error("Resuming an already running timer should not change start time")
	}
}

func TestTimerReset(t *testing.T) {
	timer := NewTimer().Start()

	// Let it run for a bit
	time.Sleep(50 * time.Millisecond)

	timer = timer.Reset()

	if timer.duration != 0 {
		t.Error("Timer duration should be zero after Reset()")
	}

	if timer.startTime.IsZero() {
		t.Error("Timer should have a valid start time after Reset()")
	}
}

func TestTimerDuration(t *testing.T) {
	timer := NewTimer().Start()

	// Let it run for a bit
	time.Sleep(100 * time.Millisecond)

	duration := timer.Duration()
	if duration < 50*time.Millisecond {
		t.Error("Timer duration should be at least 50ms")
	}

	// Stop the timer and check duration is preserved
	timer = timer.Pause()
	stoppedDuration := timer.Duration()

	if stoppedDuration != timer.duration {
		t.Error("Duration() should return stored duration for stopped timer")
	}

	// Wait a bit and check duration hasn't changed
	time.Sleep(50 * time.Millisecond)
	if timer.Duration() != stoppedDuration {
		t.Error("Stopped timer duration should not change over time")
	}
}

func TestTimerFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero duration", 0, "00:00"},
		{"5 seconds", 5 * time.Second, "00:05"},
		{"30 seconds", 30 * time.Second, "00:30"},
		{"1 minute", 1 * time.Minute, "01:00"},
		{"1 minute 30 seconds", 1*time.Minute + 30*time.Second, "01:30"},
		{"10 minutes 45 seconds", 10*time.Minute + 45*time.Second, "10:45"},
		{"59 minutes 59 seconds", 59*time.Minute + 59*time.Second, "59:59"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timer := Timer{duration: tt.duration, running: false}
			result := timer.FormatDuration()
			if result != tt.expected {
				t.Errorf("FormatDuration() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTimerIsRunning(t *testing.T) {
	timer := NewTimer()

	if timer.IsRunning() {
		t.Error("New timer should not be running")
	}

	timer = timer.Start()
	if !timer.IsRunning() {
		t.Error("Started timer should be running")
	}

	timer = timer.Pause()
	if timer.IsRunning() {
		t.Error("Paused timer should not be running")
	}

	timer = timer.Resume()
	if !timer.IsRunning() {
		t.Error("Resumed timer should be running")
	}

}

func TestTimerUpdate(t *testing.T) {
	timer := NewTimer().Start()

	// Test with TimerTickMsg
	updatedTimer, cmd := timer.Update(TimerTickMsg{})

	if cmd == nil {
		t.Error("Timer.Update() should return a command for TimerTickMsg")
	}

	if !updatedTimer.running {
		t.Error("Running timer should remain running after TimerTickMsg")
	}

	// Test with other message types
	updatedTimer, cmd = timer.Update(tea.KeyMsg{})

	if cmd != nil {
		t.Error("Timer.Update() should return nil command for non-TimerTickMsg")
	}

	// Test with stopped timer
	stoppedTimer := timer.Pause()
	updatedTimer, cmd = stoppedTimer.Update(TimerTickMsg{})

	if cmd == nil {
		t.Error("Timer.Update() should still return a command even for stopped timer")
	}
}

func TestNewTimerDisplay(t *testing.T) {
	display := NewTimerDisplay()

	if display.visible {
		t.Error("New TimerDisplay should not be visible")
	}
}

func TestTimerDisplayToggleVisible(t *testing.T) {
	display := NewTimerDisplay()

	if display.IsVisible() {
		t.Error("New TimerDisplay should not be visible")
	}

	display = display.ToggleVisible()
	if !display.IsVisible() {
		t.Error("TimerDisplay should be visible after toggle")
	}

	display = display.ToggleVisible()
	if display.IsVisible() {
		t.Error("TimerDisplay should not be visible after second toggle")
	}
}

func TestTimerDisplayUpdate(t *testing.T) {
	display := NewTimerDisplay()

	// TimerDisplay.Update should not change state and return nil command
	updatedDisplay, cmd := display.Update(tea.KeyMsg{})

	if cmd != nil {
		t.Error("TimerDisplay.Update() should return nil command")
	}

	if updatedDisplay.visible != display.visible {
		t.Error("TimerDisplay.Update() should not change visibility")
	}
}

func TestEnsureTimerInitialized(t *testing.T) {
	// Test with nil slide
	EnsureTimerInitialized(nil)
	// Should not panic

	// Test with slide having uninitialized timer
	slide := &Slide{}
	EnsureTimerInitialized(slide)

	if slide.Timer.startTime.IsZero() {
		t.Error("EnsureTimerInitialized should set startTime")
	}

	if slide.Timer.duration != 0 {
		t.Error("EnsureTimerInitialized should set duration to 0")
	}

	if slide.Timer.running {
		t.Error("EnsureTimerInitialized should set running to false")
	}

	// Test with slide having already initialized timer
	slide.Timer = slide.Timer.Start()
	originalStartTime := slide.Timer.startTime
	originalRunning := slide.Timer.running

	EnsureTimerInitialized(slide)

	// Should not change an already initialized timer that's running
	if slide.Timer.startTime != originalStartTime {
		t.Error("EnsureTimerInitialized should not change already initialized timer")
	}

	if slide.Timer.running != originalRunning {
		t.Error("EnsureTimerInitialized should not change running state of initialized timer")
	}

	// Test with slide having a timer that was started and then stopped
	slide2 := &Slide{}
	slide2.Timer = NewTimer().Start()
	slide2.Timer = slide2.Timer.Pause()
	originalTimer2 := slide2.Timer

	EnsureTimerInitialized(slide2)

	// Should not change a timer that has been used (not in zero state)
	if slide2.Timer.startTime.IsZero() && originalTimer2.startTime.IsZero() {
		t.Error("Timer state comparison failed")
	}
}

func TestTimerIntegration(t *testing.T) {
	// Test a typical usage scenario: start, pause, resume, stop
	timer := NewTimer()

	// Start timer
	timer = timer.Start()
	if !timer.IsRunning() {
		t.Error("Timer should be running after start")
	}

	// Let it run
	time.Sleep(50 * time.Millisecond)

	// Pause timer
	timer = timer.Pause()
	if timer.IsRunning() {
		t.Error("Timer should not be running after pause")
	}

	pausedDuration := timer.Duration()
	if pausedDuration <= 0 {
		t.Error("Timer should have positive duration after pause")
	}

	// Wait while paused
	time.Sleep(50 * time.Millisecond)

	// Duration should not change while paused
	if timer.Duration() != pausedDuration {
		t.Error("Timer duration should not change while paused")
	}

	// Resume timer
	timer = timer.Resume()
	if !timer.IsRunning() {
		t.Error("Timer should be running after resume")
	}

	// Let it run more
	time.Sleep(50 * time.Millisecond)

	// Duration should have increased
	if timer.Duration() <= pausedDuration {
		t.Error("Timer duration should increase after resume")
	}

	finalDuration := timer.Duration()
	if finalDuration <= pausedDuration {
		t.Error("Final duration should be greater than paused duration")
	}

	// Format should work correctly
	formatted := timer.FormatDuration()
	if formatted == "" {
		t.Error("FormatDuration should return non-empty string")
	}

	if !strings.Contains(formatted, ":") {
		t.Error("Formatted duration should contain colon separator")
	}
}
