package worker

import (
	"context"
	"errors"
	"sync"

	"classroom-attendance/go-backend/internal/models"
)

var ErrQueueFull = errors.New("frame processing queue is full")

type FrameProcessor interface {
	Process(context.Context, models.FrameRequest) (models.FrameResponse, error)
	Identify(context.Context, models.FrameRequest) (models.FrameResponse, error)
}

type job struct {
	ctx      context.Context
	request  models.FrameRequest
	identify bool
	response chan result
}

type result struct {
	value models.FrameResponse
	err   error
}

type Pool struct {
	processor FrameProcessor
	jobs      chan job
	workers   int
	wg        sync.WaitGroup
}

func NewPool(processor FrameProcessor, workers int, queueSize int) *Pool {
	if workers < 1 {
		workers = 1
	}
	if workers > 4 {
		workers = 4
	}
	if queueSize < workers {
		queueSize = workers
	}
	return &Pool{
		processor: processor,
		jobs:      make(chan job, queueSize),
		workers:   workers,
	}
}

func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case item := <-p.jobs:
					var value models.FrameResponse
					var err error
					if item.identify {
						value, err = p.processor.Identify(item.ctx, item.request)
					} else {
						value, err = p.processor.Process(item.ctx, item.request)
					}
					item.response <- result{value: value, err: err}
				}
			}
		}()
	}
}

func (p *Pool) Submit(ctx context.Context, request models.FrameRequest) (models.FrameResponse, error) {
	return p.submit(ctx, request, false)
}

func (p *Pool) SubmitIdentify(ctx context.Context, request models.FrameRequest) (models.FrameResponse, error) {
	return p.submit(ctx, request, true)
}

func (p *Pool) submit(ctx context.Context, request models.FrameRequest, identify bool) (models.FrameResponse, error) {
	response := make(chan result, 1)
	item := job{
		ctx:      ctx,
		request:  request,
		identify: identify,
		response: response,
	}

	select {
	case p.jobs <- item:
	default:
		return models.FrameResponse{}, ErrQueueFull
	}

	select {
	case <-ctx.Done():
		return models.FrameResponse{}, ctx.Err()
	case result := <-response:
		return result.value, result.err
	}
}

func (p *Pool) WorkerCount() int {
	return p.workers
}
