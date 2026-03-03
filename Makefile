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