import torch
import pandas as pd
from transformers import AutoTokenizer, AutoModelForSequenceClassification, Trainer, TrainingArguments
from datasets import Dataset

print("🚀 Starting the Argos AI system...")

# 1. Detect Mac processor (Apple Silicon MPS or CPU)
device = torch.device("mps" if torch.backends.mps.is_available() else "cpu")
print(f"⚡ Using hardware: {device}")

# 2. Download model and tokenizer
model_name = "neuralmind/bert-base-portuguese-cased"
print("📥 Downloading the BERTimbau model (this might take a minute on the first run)...")
tokenizer = AutoTokenizer.from_pretrained(model_name)
model = AutoModelForSequenceClassification.from_pretrained(model_name, num_labels=8).to(device)

# 3. Mock Data (INPI Texts)
print("📊 Injecting test data...")
mock_data = {
    "text": [
        "Método e sistema para processamento de transações financeiras utilizando blockchain e criptografia.",
        "Composição farmacêutica contendo paracetamol e cafeína para alívio de dores crônicas."
    ],
    "label": [7, 0] # Fake categories just for testing
}
df = pd.DataFrame(mock_data)
dataset = Dataset.from_pandas(df)

def tokenize_function(examples):
    return tokenizer(examples["text"], padding="max_length", truncation=True, max_length=128)

tokenized_datasets = dataset.map(tokenize_function, batched=True)

# 4. Training setup
training_args = TrainingArguments(
    output_dir="./argos_model",
    num_train_epochs=2,
    per_device_train_batch_size=2,
    report_to="none"
)

trainer = Trainer(model=model, args=training_args, train_dataset=tokenized_datasets)

print("🧠 Starting training on Mac...")
trainer.train()

# 5. Save the trained model
trainer.save_model("./argos_model")
tokenizer.save_pretrained("./argos_model")
print("✅ Argos brain successfully trained and saved in the 'argos_model' folder!")
