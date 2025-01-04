package options

// Option is a function that modifies some options type T
type Option[T any] interface {
	ApplyOption(*T) error
}

// OptionFunc is a helper to convert functions to Option interface
type OptionFunc[T any] func(*T) error

func (f OptionFunc[T]) ApplyOption(o *T) error {
	return f(o)
}

// Apply applies a list of options to a target
func Apply[T any](target *T, opts ...Option[T]) error {
	for _, opt := range opts {
		if err := opt.ApplyOption(target); err != nil {
			return err
		}
	}
	return nil
}
