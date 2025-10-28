#!/bin/bash

COMPANIES=("Tesla:TSLA" "Apple:AAPL" "Google:GOOGL" "Microsoft:MSFT" "Amazon:AMZN")

echo "Running daily backtests..."

for entry in "${COMPANIES[@]}"; do
  IFS=':' read -r company ticker <<<"$entry"

  echo "Testing $company ($ticker)..."
  curl -s "http://localhost:8080/backtest?company=$company&ticker=$ticker" | jq .

  sleep 2
done

echo "Backtests complete!"
