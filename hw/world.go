package hw

import (
	"log"
	"time"
)

type worldPrio int

const (
	PrioInput  = 0
	PrioOutput = 1
)

type Runner interface {
	Run(w *World)
}

type RunnerFunc func(w *World)

func (f RunnerFunc) Run(w *World) {
	f(w)
}

type worldEvent struct {
	time time.Duration
	prio worldPrio
	run  Runner
}

type worldEventsQueue []worldEvent

func (q worldEventsQueue) Len() int {
	return len(q)
}

func (q worldEventsQueue) Less(i, j int) bool {
	a, b := q[i], q[j]
	if a.time != b.time {
		return a.time < b.time
	}
	return a.prio < b.prio
}

func (q worldEventsQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}

func (q worldEventsQueue) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || !q.Less(j, i) {
			break
		}
		q.Swap(i, j)
		j = i
	}
}

func (q worldEventsQueue) down(i, n int) {
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && !q.Less(j1, j2) {
			j = j2 // = 2*i + 2  // right child
		}
		if !q.Less(j, i) {
			break
		}
		q.Swap(i, j)
		i = j
	}
}

func (q *worldEventsQueue) Push(e worldEvent) {
	*q = append(*q, e)
	q.up(q.Len() - 1)
}

func (q *worldEventsQueue) Pop() *worldEvent {
	n := q.Len() - 1

	q.Swap(0, n)
	q.down(0, n)

	e := &(*q)[n]
	*q = (*q)[:n]
	return e
}

type World struct {
	start time.Time
	time  time.Duration
	queue worldEventsQueue
}

func (w *World) At(t time.Duration, prio worldPrio, r Runner) {
	w.queue.Push(worldEvent{
		time: t,
		prio: prio,
		run:  r,
	})
}

func (w *World) AtStart(t time.Time, r Runner) {
	if w.start.IsZero() || t.Before(w.start) {
		offset := w.start.Sub(t)
		w.start = t
		for i := range w.queue {
			w.queue[i].time += offset
		}
	}
	w.At(0, PrioInput, r)
}

func (w *World) StartTime() time.Time {
	return w.start
}

func (w *World) Time() time.Duration {
	return w.time
}

func (w *World) Simulate() {
	var nextReport time.Time
	var count int64
	var lastCountReset time.Time

	var realStartTime, realEndTime time.Time
	var simulatedStartTime, simulatedEndTime time.Time

	realStartTime = time.Now()
	for len(w.queue) != 0 {
		e := w.queue.Pop()

		if e.time < w.time {
			panic("event time < world time")
		}
		w.time = e.time

		if count >= 10000000 {
			now := time.Now()
			if !now.Before(nextReport) {
				nextReport = now.Add(time.Second)

				if !lastCountReset.IsZero() {
					duration := now.Sub(lastCountReset)
					log.Printf(
						"simulated world time: %s (events per second: %d)",
						w.start.Local().Add(w.time).Format("2006-01-02T15:04:05.000Z07:00"),
						count*int64(time.Second)/int64(duration),
					)
				}
				count = 0
				lastCountReset = now
			}
		}

		e.run.Run(w)
		count++
	}
	simulatedStartTime = w.start
	simulatedEndTime = w.start.Add(w.time)
	realEndTime = time.Now()

	log.Println("simulated world time: -- done --", w.time)

	log.Printf(
		"%s simulated in %s",
		simulatedEndTime.Sub(simulatedStartTime),
		realEndTime.Sub(realStartTime),
	)
}
