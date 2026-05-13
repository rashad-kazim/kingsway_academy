# Workflow State Readiness

Status: placeholder before owner workflow meeting, 2026-05-13.

No final state-machine implementation is added yet because the exact owner rules are not confirmed. This document only reserves the areas that must not be implemented as loose one-off booleans/strings later.

## Areas Requiring Explicit Transition Rules

### Student lifecycle

Likely future states:

- trial
- active
- left
- graduated

Open owner questions:

- How many trial lessons are allowed?
- Does the system auto-promote trial to active?
- What happens to payments during trial?
- Can a left student return to active?
- Does branch transfer reset the registration date?

### Lesson lifecycle

Likely future states:

- planned
- started
- attendance_submitted
- finished
- cancelled

Open owner questions:

- Who can start/end a lesson?
- Can a lesson finish without attendance?
- What happens if teacher forgets to finish?
- Should salary be locked at finish time?

### Attendance lifecycle

Likely future states:

- pending
- present
- absent
- excused
- late

Open owner questions:

- Can receptionist edit teacher attendance?
- Is parent notification required for absent students?
- Does attendance affect teacher salary?

### Payment lifecycle

Likely future states:

- pending
- partially_paid
- paid
- overdue
- cancelled
- refunded

Open owner questions:

- Are partial payments allowed?
- Does overdue block attendance/exams?
- Who can cancel or refund?

### Teacher assignment / swap lifecycle

Likely future states:

- assigned
- swap_pending
- swapped
- cancelled

Open owner questions:

- Does swap happen immediately or from selected date?
- Who approves swap?
- How salary is split on swap day?

### Exam lifecycle

Likely future states:

- draft
- scheduled
- active
- submitted
- grading
- graded
- archived

Open owner questions:

- Can students resume interrupted online exams?
- Are speaking recordings mandatory?
- Who can reopen an exam?

## Rule Before Coding

For any of the areas above, write the transition table first:

| From | Event | To | Actor | Guards | Side effects |
| --- | --- | --- | --- | --- | --- |

Then implement:

1. DB constraint or state table.
2. Service-level transition function.
3. Audit log.
4. Idempotent write endpoint.
5. Integration test for invalid transitions.
