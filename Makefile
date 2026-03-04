# Define the examples as a list
EXAMPLES := animation counter demo routing scrollables shapes

.PHONY: run bench $(EXAMPLES)

# The 'run' command now depends on each individual example
run: $(EXAMPLES)

# This target runs each example in the background
$(EXAMPLES):
	go run ./examples/$@ &

# The bench stays largely the same, but you can add -cpu if needed
bench:
	go test -bench=. -benchmem -cpu=4

run-anim:
	go run ./examples/animation
run-app:
	go run ./examples/app
run-counter:
	go run ./examples/counter
run-demo:
	go run ./examples/demo
run-routing:
	go run ./examples/routing
run-scrollables:
	go run ./examples/scrollables
run-shapes:
	go run ./examples/shapes