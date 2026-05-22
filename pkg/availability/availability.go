package availability

import "time"

const (
	StatusOpen    = "open"
	StatusPending = "pending"
	StatusClosed  = "closed"
	StatusUnknown = "unknown"

	SourceApplicationNotice = "application_notice"
	SourceSchedule          = "schedule"
	SourceNone              = "none"
)

type ScheduleEvidence struct {
	StartsAt time.Time
	EndsAt   time.Time
}

type ApplicationEvidence struct {
	Status string
}

type Decision struct {
	Status            string
	Source            string
	ScheduleStatus    string
	ApplicationStatus string
	Conflict          bool
}

func Resolve(now time.Time, schedules []ScheduleEvidence, applications []ApplicationEvidence) Decision {
	scheduleStatus := scheduleAvailability(now, schedules)
	applicationStatus := applicationAvailability(applications)
	status := StatusUnknown
	source := SourceNone

	if applicationStatus != StatusUnknown {
		status = applicationStatus
		source = SourceApplicationNotice
	} else if scheduleStatus != StatusUnknown {
		status = scheduleStatus
		source = SourceSchedule
	}

	return Decision{
		Status:            status,
		Source:            source,
		ScheduleStatus:    scheduleStatus,
		ApplicationStatus: applicationStatus,
		Conflict:          hasConflict(scheduleStatus, applicationStatus),
	}
}

func scheduleAvailability(now time.Time, schedules []ScheduleEvidence) string {
	if len(schedules) == 0 {
		return StatusUnknown
	}
	hasFuture := false
	for _, schedule := range schedules {
		if schedule.StartsAt.IsZero() || schedule.EndsAt.IsZero() {
			continue
		}
		if !now.Before(schedule.StartsAt) && !now.After(schedule.EndsAt) {
			return StatusOpen
		}
		if now.Before(schedule.StartsAt) {
			hasFuture = true
		}
	}
	if hasFuture {
		return StatusPending
	}
	return StatusClosed
}

func applicationAvailability(applications []ApplicationEvidence) string {
	status := StatusUnknown
	for _, application := range applications {
		switch application.Status {
		case "청약중":
			return StatusOpen
		case "접수예정":
			status = StatusPending
		}
	}
	return status
}

func hasConflict(scheduleStatus string, applicationStatus string) bool {
	if scheduleStatus == StatusUnknown || applicationStatus == StatusUnknown {
		return false
	}
	return scheduleStatus != applicationStatus
}
