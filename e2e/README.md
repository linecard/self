# E2E Tests

## Usage

`just`

## Psuedocode

### Setup
```bash
for scaffold in ruby python node go; do
    self init $scaffold init/testing-$scaffold
    self publish init/testing-$scaffold
    self deploy init/testing-$scaffold
done
```
approximates `just setup`

### Test
```bash
for scaffold in ruby python node go; do
    FUNCTION_PATH=init/testing-$scaffold bundle exec -- rspec -f d
done
```
approximates `just test`

## Teardown
```bash
for scaffold in ruby python node go; do
    self destroy init/testing-$scaffold
done
```

## Ensure and Config State Mutations
All just commands can be given one or many `-E config/<name>.env` files to permute state/tests.

For example, if we wanted to test the correct deployment state is achieved when `self` is instructed to deploy to an API gateway, we pass the `-E config/gateway.env` file to all `just` commands. If we want to mutate our current test deployments to a new state, we can do so mid-test-cycle via `just -E config/<desired>.env ensure`.