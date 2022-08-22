package async

import (
	"context"

	"github.com/41north/web3/pkg/util"
)

type Future[T any] interface {
	Result() <-chan util.Result[*T]

	Await(ctx context.Context) (*T, error)
}

func NewFuture[T any](ch <-chan util.Result[*T]) Future[T] {
	return future[T]{
		result: ch,
	}
}

func NewFutureFailed[T any](errors ...error) Future[T] {
	ch := make(chan util.Result[*T], len(errors))
	for _, err := range errors {
		ch <- util.NewResultErr[*T](err)
	}
	close(ch)
	return NewFuture[T](ch)
}

func NewFutureImmediate[T any](results ...util.Result[*T]) Future[T] {
	ch := make(chan util.Result[*T], len(results))
	for _, result := range results {
		ch <- result
	}
	close(ch)
	return NewFuture[T](ch)
}

type future[T any] struct {
	result <-chan util.Result[*T]
}

func (f future[T]) Result() <-chan util.Result[*T] {
	return f.result
}

func (f future[T]) Await(ctx context.Context) (*T, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-f.result:
		return result.Value()
	}
}

func Map[A any, B any](
	ctx context.Context,
	future Future[A],
	mapper func(value A) (*B, error),
) Future[B] {

	inChannel := future.Result()
	outChannel := make(chan util.Result[*B])

	go func() {
		for {
			select {
			case <-ctx.Done():
				outChannel <- util.NewResultErr[*B](ctx.Err())
				close(outChannel)

			case result, ok := <-inChannel:

				if !ok && result == nil {
					// channel was closed before anything was sent to it
					close(outChannel)
					return
				}

				value, err := result.Value()
				if err != nil {
					outChannel <- util.NewResultErr[*B](err)
				}
				if value != nil {
					mapped, err := mapper(*value)
					if err != nil {
						outChannel <- util.NewResultErr[*B](err)
					}

					outChannel <- util.NewResult[*B](mapped)

				}

				if !ok {
					// upstream channel has been closed
					close(outChannel)
				}
			}
		}
	}()

	return NewFuture(outChannel)
}
