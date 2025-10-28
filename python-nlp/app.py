from flask import Flask, request, jsonify
from transformers import AutoTokenizer, AutoModelForSequenceClassification
import torch
import logging

app = Flask(__name__)
logging.basicConfig(level=logging.INFO)

# Load FinBERT model
MODEL_NAME = "ProsusAI/finbert"
tokenizer = AutoTokenizer.from_pretrained(MODEL_NAME)
model = AutoModelForSequenceClassification.from_pretrained(MODEL_NAME)


@app.route("/health", methods=["GET"])
def health():
    return jsonify({"status": "healthy", "model": MODEL_NAME})


@app.route("/analyze", methods=["POST"])
def analyze():
    try:
        data = request.get_json()

        if not data or "text" not in data:
            return jsonify({"error": "Missing 'text' field"}), 400

        text = data["text"]

        # Truncate to avoid token limit
        if len(text) > 512:
            text = text[:512]

        # Tokenize and analyze
        inputs = tokenizer(text, return_tensors="pt", truncation=True, max_length=512)

        with torch.no_grad():
            outputs = model(**inputs)
            predictions = torch.nn.functional.softmax(outputs.logits, dim=1)

        # FinBERT returns: [positive, negative, neutral]
        scores = predictions[0].tolist()

        result = {"positive": scores[0], "negative": scores[1], "neutral": scores[2]}

        logging.info(f"Analysis complete: {result}")
        return jsonify(result)

    except Exception as e:
        logging.error(f"Analysis error: {str(e)}")
        return jsonify({"error": str(e)}), 500


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8000, debug=True)
