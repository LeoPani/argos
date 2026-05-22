from fastapi import FastAPI
from pydantic import BaseModel
import torch
from transformers import AutoTokenizer, AutoModelForSequenceClassification

# 1. Configurar o aplicativo FastAPI
app = FastAPI(title="Argos AI - Classificação de Patentes")

# 2. Carregar o modelo e o tokenizador que treinamos
print("🧠 Carregando o modelo do Argos...")
device = torch.device("mps" if torch.backends.mps.is_available() else "cpu")
model_path = "./argos_model"

tokenizer = AutoTokenizer.from_pretrained(model_path)
model = AutoModelForSequenceClassification.from_pretrained(model_path).to(device)
model.eval() # Coloca o modelo em modo de avaliação (não vai mais treinar)
print("✅ Modelo carregado e pronto para inferência!")

# 3. Definir a estrutura dos dados de entrada (o que o Go vai enviar)
class PatentRequest(BaseModel):
    text: str

# 4. Criar o endpoint de classificação
@app.post("/classify")
async def classify_patent(request: PatentRequest):
    # Preparar o texto recebido para a IA
    inputs = tokenizer(request.text, return_tensors="pt", truncation=True, max_length=128).to(device)
    
    # Fazer a previsão sem calcular gradientes (para ser mais rápido)
    with torch.no_grad():
        outputs = model(**inputs)
    
    # Pegar a categoria com a maior pontuação matemática
    predicted_class_id = outputs.logits.argmax(dim=-1).item()
    
    return {
        "text_received": request.text,
        "predicted_category_id": predicted_class_id
    }
