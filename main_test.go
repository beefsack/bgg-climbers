package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRatingAverage(t *testing.T) {
	assert.Equal(t, 5.0, NewRatingAverage(
		Record{
			Average:    "10",
			UsersRated: "1",
		},
		Record{
			Average:    "7.5",
			UsersRated: "2",
		},
	))

	assert.Equal(t, 7.0, NewRatingAverage(
		Record{
			Average:    "7",
			UsersRated: "1",
		},
		Record{
			Average:    "7",
			UsersRated: "2",
		},
	))

	assert.Equal(t, 7.0, NewRatingAverage(
		Record{
			Average:    "6",
			UsersRated: "2",
		},
		Record{
			Average:    "7",
			UsersRated: "2",
		},
	))
}
