## About

Stripe subscriptions can now have multiple plans per subscription. Many of you
have multiple subscriptions per customer. This tool will combine those
subscriptions into a single subscription. Now your customer will only be
charged once!

## Install

The easiest way to get `submigrate` is to download a pre-built binary.

- OS X
- Windows
- Linux

You can also install using the Go tool.

```
go get github.com/stripe/submigrate
```

## Usage

`submigrate` takes a list of subscription IDs. By default, no actions are
taken, so it's safe to run. You'll see logging explaining what would have
happened.

```
$ submigrate -h
Usage of submigrate:
  -key string
        Stripe API key
  -run
        Combine the subscriptions
```

## Migration Strategy
