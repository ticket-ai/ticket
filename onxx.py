from transformers import AutoTokenizer, AutoModel, AutoConfig
import torch


# 1) clone the repo and then point to this to the folder
model_dir = "~/guardian/thefolderwiththepytorchbin"

config = AutoConfig.from_pretrained("bert-base-uncased")
config.save_pretrained(model_dir)

print(f"✅ Wrote config.json to {model_dir}")

# 2) load
tokenizer = AutoTokenizer.from_pretrained(model_dir)
model = AutoModel.from_pretrained(model_dir)
model.eval()

# 3) create a dummy input (adjust max_length as needed)
text = "Hello, world!"
inputs = tokenizer(text, return_tensors="pt", padding="max_length", max_length=32, truncation=True)

# 4) export
torch.onnx.export(
    model,
    (inputs["input_ids"], inputs["attention_mask"]),
    "model.onnx",
    input_names=["input_ids", "attention_mask"],
    output_names=["last_hidden_state"],
    dynamic_axes={
        "input_ids": {0: "batch", 1: "seq"},
        "attention_mask": {0: "batch", 1: "seq"},
        "last_hidden_state": {0: "batch", 1: "seq"},
    },
    opset_version=14,
)

print("✅ Exported ONNX model")