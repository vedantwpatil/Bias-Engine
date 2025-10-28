import yfinance as yf
from flask import Flask, request, jsonify
from datetime import datetime, timedelta
import logging

app = Flask(__name__)
logging.basicConfig(level=logging.INFO)


@app.route("/health", methods=["GET"])
def health():
    return jsonify({"status": "healthy", "service": "stock_fetcher"})


@app.route("/stock_data", methods=["POST"])
def get_stock_data():
    """Get historical stock data for a ticker"""
    try:
        data = request.get_json()
        ticker = data["ticker"]
        start_date = data.get("start_date")
        end_date = data.get("end_date")

        logging.info(f"Fetching {ticker} from {start_date} to {end_date}")

        stock = yf.Ticker(ticker)
        hist = stock.history(start=start_date, end=end_date)

        if hist.empty:
            return jsonify({"error": f"No data found for {ticker}"}), 404

        result = []
        for date, row in hist.iterrows():
            result.append(
                {
                    "date": date.strftime("%Y-%m-%d"),
                    "open": float(row["Open"]),
                    "close": float(row["Close"]),
                    "high": float(row["High"]),
                    "low": float(row["Low"]),
                    "volume": int(row["Volume"]),
                }
            )

        logging.info(f"Returned {len(result)} data points for {ticker}")
        return jsonify(result)

    except Exception as e:
        logging.error(f"Error: {str(e)}")
        return jsonify({"error": str(e)}), 500


@app.route("/stock_price", methods=["POST"])
def get_stock_price():
    """Get stock price for a specific date"""
    try:
        data = request.get_json()
        ticker = data["ticker"]
        date = data["date"]  # Format: YYYY-MM-DD

        # Fetch data around the date (in case market was closed)
        stock = yf.Ticker(ticker)
        start = datetime.strptime(date, "%Y-%m-%d") - timedelta(days=5)
        end = datetime.strptime(date, "%Y-%m-%d") + timedelta(days=5)

        hist = stock.history(
            start=start.strftime("%Y-%m-%d"), end=end.strftime("%Y-%m-%d")
        )

        if hist.empty:
            return jsonify({"error": f"No data for {ticker} on {date}"}), 404

        # Get closest date
        target_date = datetime.strptime(date, "%Y-%m-%d")
        closest_idx = min(
            range(len(hist)),
            key=lambda i: abs((hist.index[i].to_pydatetime() - target_date).days),
        )

        row = hist.iloc[closest_idx]

        result = {
            "ticker": ticker,
            "requested_date": date,
            "actual_date": hist.index[closest_idx].strftime("%Y-%m-%d"),
            "open": float(row["Open"]),
            "close": float(row["Close"]),
            "high": float(row["High"]),
            "low": float(row["Low"]),
        }

        return jsonify(result)

    except Exception as e:
        logging.error(f"Error: {str(e)}")
        return jsonify({"error": str(e)}), 500


@app.route("/company_info", methods=["POST"])
def get_company_info():
    """Get company information"""
    try:
        data = request.get_json()
        ticker = data["ticker"]

        stock = yf.Ticker(ticker)
        info = stock.info

        result = {
            "ticker": ticker,
            "name": info.get("longName", "Unknown"),
            "sector": info.get("sector", "Unknown"),
            "industry": info.get("industry", "Unknown"),
            "market_cap": info.get("marketCap", 0),
        }

        return jsonify(result)

    except Exception as e:
        return jsonify({"error": str(e)}), 500


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8001, debug=True)
