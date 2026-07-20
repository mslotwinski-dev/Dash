package middlewares

import (
	"time"
)

type CircuitBreaker struct {
	MaxFailures int
	Timeout     time.Duration
}

func NewCircuitBreaker(maxFailures int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		MaxFailures: maxFailures,
		Timeout:     timeout,
	}
}

// Ten mechanizm pozwala opakować logikę wywołania. 
// Ze względu na uproszczenie w tej wersji, logika jest zintegrowana bezpośrdenio w reverse proxy i backend.
