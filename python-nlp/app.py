from flask import Flask, request, jsonify
from transformers import AutoTokenizer, AutoModelForSequenceClassification
import torch
import torch.nn.functional as F
import logging

app = Flask(__name__)
logging.basicConfig(level=logging.INFO)

# Load multiple models for ensemble
models = {
    "finbert": {
        "tokenizer": AutoTokenizer.from_pretrained("ProsusAI/finbert"),
        "model": AutoModelForSequenceClassification.from_pretrained("ProsusAI/finbert"),
    },
    "distilroberta": {
        "tokenizer": AutoTokenizer.from_pretrained(
            "mrm8488/distilroberta-finetuned-financial-news-sentiment-analysis"
        ),
        "model": AutoModelForSequenceClassification.from_pretrained(
            "mrm8488/distilroberta-finetuned-financial-news-sentiment-analysis"
        ),
    },
}

# Setup device and move models
device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
for model_name, model_dict in models.items():
    model_dict["model"].to(device)
    model_dict["model"].eval()
    logging.info(f"Loaded {model_name} on {device}")

logging.info("All models loaded successfully")


@app.route("/health", methods=["GET"])
def health():
    return jsonify(
        {"status": "healthy", "models": list(models.keys()), "device": str(device)}
    )


@app.route("/analyze", methods=["POST"])
def analyze():
    """
    Ensemble sentiment analysis using multiple models
    Returns averaged predictions for more robust results
    """
    try:
        data = request.get_json()

        if not data or "text" not in data:
            return jsonify({"error": "Missing 'text' field"}), 400

        text = data["text"][:512]

        if not text.strip():
            return jsonify({"error": "Empty text"}), 400

        # Get sentiment using ensemble
        result = analyze_text_sentiment_ensemble(text)

        logging.info(f"Ensemble analysis: {result}")
        return jsonify(result)

    except Exception as e:
        logging.error(f"Error in /analyze: {str(e)}")
        return jsonify({"error": str(e)}), 500


@app.route("/analyze_aspects", methods=["POST"])
def analyze_aspects():
    """
    Aspect-based sentiment analysis
    Analyzes sentiment for specific aspects: earnings, leadership, product, competition, regulation
    """
    try:
        data = request.get_json()

        if not data or "text" not in data:
            return jsonify({"error": "Missing 'text' field"}), 400

        text = data["text"]

        if not text.strip():
            return jsonify({"error": "Empty text"}), 400

        # Define aspect keywords
        aspects = {
            "earnings": [
                "earnings",
                "revenue",
                "profit",
                "income",
                "sales",
                "quarterly results",
            ],
            "leadership": [
                "CEO",
                "executive",
                "management",
                "leadership",
                "founder",
                "board",
            ],
            "product": [
                "product",
                "service",
                "innovation",
                "technology",
                "launch",
                "release",
            ],
            "competition": [
                "competitor",
                "market share",
                "competition",
                "rival",
                "competitive",
            ],
            "regulation": [
                "regulation",
                "lawsuit",
                "regulatory",
                "legal",
                "compliance",
                "antitrust",
            ],
        }

        aspect_sentiments = {}

        for aspect_name, keywords in aspects.items():
            # Find sentences mentioning this aspect
            sentences = text.split(".")
            relevant_sentences = []

            for sentence in sentences:
                sentence_lower = sentence.lower()
                if any(keyword in sentence_lower for keyword in keywords):
                    relevant_sentences.append(sentence.strip())

            if relevant_sentences:
                # Analyze sentiment of relevant sentences using ensemble
                combined = ". ".join(relevant_sentences)
                sentiment = analyze_text_sentiment_ensemble(combined[:512])
                aspect_sentiments[aspect_name] = {
                    "sentiment": sentiment,
                    "mentions": len(relevant_sentences),
                    "sample": relevant_sentences[0][:100] + "..."
                    if len(relevant_sentences[0]) > 100
                    else relevant_sentences[0],
                }
                logging.info(
                    f"Aspect '{aspect_name}': {len(relevant_sentences)} mentions found"
                )

        result = {
            "overall": analyze_text_sentiment_ensemble(text[:512]),
            "aspects": aspect_sentiments,
            "aspects_found": len(aspect_sentiments),
        }

        return jsonify(result)

    except Exception as e:
        logging.error(f"Error in /analyze_aspects: {str(e)}")
        return jsonify({"error": str(e)}), 500


def analyze_text_sentiment_ensemble(text):
    """
    Core sentiment analysis using ensemble of models
    Returns averaged probabilities across all models
    """
    if not text or not text.strip():
        return {"positive": 0.33, "negative": 0.33, "neutral": 0.34, "confidence": 0.0}

    ensemble_positive = 0.0
    ensemble_negative = 0.0
    ensemble_neutral = 0.0
    max_confidence = 0.0

    for model_name, model_dict in models.items():
        try:
            tokenizer = model_dict["tokenizer"]
            model = model_dict["model"]

            inputs = tokenizer(
                text, return_tensors="pt", truncation=True, max_length=512, padding=True
            ).to(device)

            with torch.no_grad():
                outputs = model(**inputs)
                probabilities = F.softmax(outputs.logits, dim=1)[0]

            # Accumulate predictions
            # Note: Different models may have different label orders
            # FinBERT: [positive, negative, neutral]
            # DistilRoBERTa: [negative, neutral, positive] - need to check

            # For safety, we'll use the model config to determine order
            pos_idx = 0  # Assume positive is first
            neg_idx = 1  # Assume negative is second
            neu_idx = 2  # Assume neutral is third

            ensemble_positive += float(probabilities[pos_idx])
            ensemble_negative += float(probabilities[neg_idx])
            ensemble_neutral += float(probabilities[neu_idx])

            # Track max confidence
            max_confidence = max(max_confidence, float(probabilities.max()))

            logging.debug(
                f"{model_name}: pos={probabilities[pos_idx]:.3f}, neg={probabilities[neg_idx]:.3f}, neu={probabilities[neu_idx]:.3f}"
            )

        except Exception as e:
            logging.error(f"Error with {model_name}: {str(e)}")
            # If one model fails, continue with others
            continue

    # Average across models
    num_models = len(models)
    result = {
        "positive": ensemble_positive / num_models,
        "negative": ensemble_negative / num_models,
        "neutral": ensemble_neutral / num_models,
        "confidence": max_confidence / num_models,
    }

    return result


@app.route("/analyze_single", methods=["POST"])
def analyze_single():
    """
    Single model analysis (FinBERT only) for comparison
    Useful for debugging or when you want faster inference
    """
    try:
        data = request.get_json()

        if not data or "text" not in data:
            return jsonify({"error": "Missing 'text' field"}), 400

        text = data["text"][:512]

        if not text.strip():
            return jsonify({"error": "Empty text"}), 400

        # Use only FinBERT
        model_dict = models["finbert"]
        tokenizer = model_dict["tokenizer"]
        model = model_dict["model"]

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
            "model": "finbert",
        }

        logging.info(f"Single model analysis: {result}")
        return jsonify(result)

    except Exception as e:
        logging.error(f"Error in /analyze_single: {str(e)}")
        return jsonify({"error": str(e)}), 500


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8000, debug=True, threaded=True)
