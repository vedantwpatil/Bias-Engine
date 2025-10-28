from flask import Flask, request, jsonify
from transformers import AutoTokenizer, AutoModelForSequenceClassification
import torch
import torch.nn.functional as F
import logging

app = Flask(__name__)
logging.basicConfig(level=logging.INFO)

MODEL_NAME = "ProsusAI/finbert"

# This approach has proper type hints
tokenizer = AutoTokenizer.from_pretrained(MODEL_NAME)
model = AutoModelForSequenceClassification.from_pretrained(MODEL_NAME)

device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
model.to(device)
model.eval()

logging.info(f"Model loaded on {device}")


@app.route("/health", methods=["GET"])
def health():
    return jsonify({"status": "healthy", "model": MODEL_NAME})


@app.route("/analyze", methods=["POST"])
def analyze():
    try:
        data = request.get_json()

        if not data or "text" not in data:
            return jsonify({"error": "Missing 'text' field"}), 400

        text = data["text"][:512]

        if not text.strip():
            return jsonify({"error": "Empty text"}), 400

        inputs = tokenizer(
            text, return_tensors="pt", truncation=True, max_length=512, padding=True
        ).to(device)

        with torch.no_grad():
            outputs = model(**inputs)
            probabilities = F.softmax(outputs.logits, dim=1)[0]

        result = {
            "positive": float(probabilities[0]),
            "negative": float(probabilities[1]),
            "neutral": float(probabilities[2]),
        }

        logging.info(f"Analysis: {result}")
        return jsonify(result)

    except Exception as e:
        logging.error(f"Error: {str(e)}")
        return jsonify({"error": str(e)}), 500


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8000, debug=True, threaded=True)
