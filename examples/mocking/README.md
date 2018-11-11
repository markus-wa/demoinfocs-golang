# Mocking the parser

This example shows you how to use the provided [`fake` package](https://godoc.org/github.com/markus-wa/demoinfocs-golang/fake) to mock `demoinfocs.IParser` and other parts of the library.
That way you will be able to write useful unit tests for your application.

## System under test

First, let's have a look at the API of our code, the 'system under test':

```go
import (
	dem "github.com/markus-wa/demoinfocs-golang"
	events "github.com/markus-wa/demoinfocs-golang/events"
)

func collectKills(parser dem.IParser) (kills []events.Kill, err error) {
    ...
}
```

We deliberately ignore the implementation so we don't make assumptions about the code since it might change in the future.

As you can see `collectKills` takes an `IParser` as input and returns a slice of `events.Kill` and potentially an error.

## Positive test case

Now let's have a look at our first test. Here we want to ensure that all kills are collected and that the order of the collected events is correct.

```go
import (
	"errors"
	"testing"

	assert "github.com/stretchr/testify/assert"

	common "github.com/markus-wa/demoinfocs-golang/common"
	events "github.com/markus-wa/demoinfocs-golang/events"
	fake "github.com/markus-wa/demoinfocs-golang/fake"
)

func TestCollectKills(t *testing.T) {
	parser := fake.NewParser()
	kill1 := kill(common.EqAK47)
	kill2 := kill(common.EqScout)
	kill3 := kill(common.EqAUG)
	parser.MockEvents(kill1)        // First frame
	parser.MockEvents(kill2, kill3) // Second frame

	parser.On("ParseToEnd").Return(nil) // Return no error

	actual, err := collectKills(parser)

	assert.Nil(t, err)
	expected := []events.Kill{kill1, kill2, kill3}
	assert.Equal(t, expected, actual)
}

func kill(wep common.EquipmentElement) events.Kill {
	eq := common.NewEquipment(wep)
	return events.Kill{
		Killer: new(common.Player),
		Weapon: &eq,
		Victim: new(common.Player),
	}
}
```

As you can see we first create a mocked parser with `fake.NewParser()`.

Then we create two `Kill` events and add them into the `Parser.Events` map.
The map index indicates at which frame the events will be sent out, in our case that's during the first and second frame, as we just iterate over the slice indices.

Note: Especially when used together with `Parser.NetMessages` it can be useful to set these indices manually to ensure the events and net-messages are sent at the right moment.

## Negative test case

Last but not least we want to do another test that ensures any error the parser encounters is returned to the callee and not suppressed by our function.

```go
import (
	"errors"
	"testing"

	assert "github.com/stretchr/testify/assert"

	common "github.com/markus-wa/demoinfocs-golang/common"
	events "github.com/markus-wa/demoinfocs-golang/events"
	fake "github.com/markus-wa/demoinfocs-golang/fake"
)

func TestCollectKillsError(t *testing.T) {
	parser := fake.NewParser()
	expectedErr := errors.New("Test error")
	parser.On("ParseToEnd").Return(expectedErr)

	kills, actualErr := collectKills(parser)

	assert.Equal(t, expectedErr, actualErr)
	assert.Nil(t, kills)
}
```

This test simply tells the mock to return the specified error and asserts that our function returns it to us.
It also makes sure that kills is nil, and not an empty slice.
