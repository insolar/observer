package tree

// Code generated by http://github.com/gojuno/minimock (2.1.9). DO NOT EDIT.

import (
	"context"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock"
	"github.com/insolar/insolar/insolar"
)

// BuilderMock implements Builder
type BuilderMock struct {
	t minimock.Tester

	funcBuild          func(ctx context.Context, reqID insolar.ID) (s1 Structure, err error)
	inspectFuncBuild   func(ctx context.Context, reqID insolar.ID)
	afterBuildCounter  uint64
	beforeBuildCounter uint64
	BuildMock          mBuilderMockBuild
}

// NewBuilderMock returns a mock for Builder
func NewBuilderMock(t minimock.Tester) *BuilderMock {
	m := &BuilderMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.BuildMock = mBuilderMockBuild{mock: m}
	m.BuildMock.callArgs = []*BuilderMockBuildParams{}

	return m
}

type mBuilderMockBuild struct {
	mock               *BuilderMock
	defaultExpectation *BuilderMockBuildExpectation
	expectations       []*BuilderMockBuildExpectation

	callArgs []*BuilderMockBuildParams
	mutex    sync.RWMutex
}

// BuilderMockBuildExpectation specifies expectation struct of the Builder.Build
type BuilderMockBuildExpectation struct {
	mock    *BuilderMock
	params  *BuilderMockBuildParams
	results *BuilderMockBuildResults
	Counter uint64
}

// BuilderMockBuildParams contains parameters of the Builder.Build
type BuilderMockBuildParams struct {
	ctx   context.Context
	reqID insolar.ID
}

// BuilderMockBuildResults contains results of the Builder.Build
type BuilderMockBuildResults struct {
	s1  Structure
	err error
}

// Expect sets up expected params for Builder.Build
func (mmBuild *mBuilderMockBuild) Expect(ctx context.Context, reqID insolar.ID) *mBuilderMockBuild {
	if mmBuild.mock.funcBuild != nil {
		mmBuild.mock.t.Fatalf("BuilderMock.Build mock is already set by Set")
	}

	if mmBuild.defaultExpectation == nil {
		mmBuild.defaultExpectation = &BuilderMockBuildExpectation{}
	}

	mmBuild.defaultExpectation.params = &BuilderMockBuildParams{ctx, reqID}
	for _, e := range mmBuild.expectations {
		if minimock.Equal(e.params, mmBuild.defaultExpectation.params) {
			mmBuild.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmBuild.defaultExpectation.params)
		}
	}

	return mmBuild
}

// Inspect accepts an inspector function that has same arguments as the Builder.Build
func (mmBuild *mBuilderMockBuild) Inspect(f func(ctx context.Context, reqID insolar.ID)) *mBuilderMockBuild {
	if mmBuild.mock.inspectFuncBuild != nil {
		mmBuild.mock.t.Fatalf("Inspect function is already set for BuilderMock.Build")
	}

	mmBuild.mock.inspectFuncBuild = f

	return mmBuild
}

// Return sets up results that will be returned by Builder.Build
func (mmBuild *mBuilderMockBuild) Return(s1 Structure, err error) *BuilderMock {
	if mmBuild.mock.funcBuild != nil {
		mmBuild.mock.t.Fatalf("BuilderMock.Build mock is already set by Set")
	}

	if mmBuild.defaultExpectation == nil {
		mmBuild.defaultExpectation = &BuilderMockBuildExpectation{mock: mmBuild.mock}
	}
	mmBuild.defaultExpectation.results = &BuilderMockBuildResults{s1, err}
	return mmBuild.mock
}

//Set uses given function f to mock the Builder.Build method
func (mmBuild *mBuilderMockBuild) Set(f func(ctx context.Context, reqID insolar.ID) (s1 Structure, err error)) *BuilderMock {
	if mmBuild.defaultExpectation != nil {
		mmBuild.mock.t.Fatalf("Default expectation is already set for the Builder.Build method")
	}

	if len(mmBuild.expectations) > 0 {
		mmBuild.mock.t.Fatalf("Some expectations are already set for the Builder.Build method")
	}

	mmBuild.mock.funcBuild = f
	return mmBuild.mock
}

// When sets expectation for the Builder.Build which will trigger the result defined by the following
// Then helper
func (mmBuild *mBuilderMockBuild) When(ctx context.Context, reqID insolar.ID) *BuilderMockBuildExpectation {
	if mmBuild.mock.funcBuild != nil {
		mmBuild.mock.t.Fatalf("BuilderMock.Build mock is already set by Set")
	}

	expectation := &BuilderMockBuildExpectation{
		mock:   mmBuild.mock,
		params: &BuilderMockBuildParams{ctx, reqID},
	}
	mmBuild.expectations = append(mmBuild.expectations, expectation)
	return expectation
}

// Then sets up Builder.Build return parameters for the expectation previously defined by the When method
func (e *BuilderMockBuildExpectation) Then(s1 Structure, err error) *BuilderMock {
	e.results = &BuilderMockBuildResults{s1, err}
	return e.mock
}

// Build implements Builder
func (mmBuild *BuilderMock) Build(ctx context.Context, reqID insolar.ID) (s1 Structure, err error) {
	mm_atomic.AddUint64(&mmBuild.beforeBuildCounter, 1)
	defer mm_atomic.AddUint64(&mmBuild.afterBuildCounter, 1)

	if mmBuild.inspectFuncBuild != nil {
		mmBuild.inspectFuncBuild(ctx, reqID)
	}

	mm_params := &BuilderMockBuildParams{ctx, reqID}

	// Record call args
	mmBuild.BuildMock.mutex.Lock()
	mmBuild.BuildMock.callArgs = append(mmBuild.BuildMock.callArgs, mm_params)
	mmBuild.BuildMock.mutex.Unlock()

	for _, e := range mmBuild.BuildMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.s1, e.results.err
		}
	}

	if mmBuild.BuildMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmBuild.BuildMock.defaultExpectation.Counter, 1)
		mm_want := mmBuild.BuildMock.defaultExpectation.params
		mm_got := BuilderMockBuildParams{ctx, reqID}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmBuild.t.Errorf("BuilderMock.Build got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmBuild.BuildMock.defaultExpectation.results
		if mm_results == nil {
			mmBuild.t.Fatal("No results are set for the BuilderMock.Build")
		}
		return (*mm_results).s1, (*mm_results).err
	}
	if mmBuild.funcBuild != nil {
		return mmBuild.funcBuild(ctx, reqID)
	}
	mmBuild.t.Fatalf("Unexpected call to BuilderMock.Build. %v %v", ctx, reqID)
	return
}

// BuildAfterCounter returns a count of finished BuilderMock.Build invocations
func (mmBuild *BuilderMock) BuildAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmBuild.afterBuildCounter)
}

// BuildBeforeCounter returns a count of BuilderMock.Build invocations
func (mmBuild *BuilderMock) BuildBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmBuild.beforeBuildCounter)
}

// Calls returns a list of arguments used in each call to BuilderMock.Build.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmBuild *mBuilderMockBuild) Calls() []*BuilderMockBuildParams {
	mmBuild.mutex.RLock()

	argCopy := make([]*BuilderMockBuildParams, len(mmBuild.callArgs))
	copy(argCopy, mmBuild.callArgs)

	mmBuild.mutex.RUnlock()

	return argCopy
}

// MinimockBuildDone returns true if the count of the Build invocations corresponds
// the number of defined expectations
func (m *BuilderMock) MinimockBuildDone() bool {
	for _, e := range m.BuildMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.BuildMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterBuildCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcBuild != nil && mm_atomic.LoadUint64(&m.afterBuildCounter) < 1 {
		return false
	}
	return true
}

// MinimockBuildInspect logs each unmet expectation
func (m *BuilderMock) MinimockBuildInspect() {
	for _, e := range m.BuildMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to BuilderMock.Build with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.BuildMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterBuildCounter) < 1 {
		if m.BuildMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to BuilderMock.Build")
		} else {
			m.t.Errorf("Expected call to BuilderMock.Build with params: %#v", *m.BuildMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcBuild != nil && mm_atomic.LoadUint64(&m.afterBuildCounter) < 1 {
		m.t.Error("Expected call to BuilderMock.Build")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *BuilderMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockBuildInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *BuilderMock) MinimockWait(timeout mm_time.Duration) {
	timeoutCh := mm_time.After(timeout)
	for {
		if m.minimockDone() {
			return
		}
		select {
		case <-timeoutCh:
			m.MinimockFinish()
			return
		case <-mm_time.After(10 * mm_time.Millisecond):
		}
	}
}

func (m *BuilderMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockBuildDone()
}