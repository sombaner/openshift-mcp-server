import torch
import torch.nn as nn

# Dummy PyTorch model for demonstration
class DummyModel(nn.Module):
    def __init__(self):
        super(DummyModel, self).__init__()
        self.linear = nn.Linear(1, 1)

    def forward(self, x):
        return self.linear(x)

if __name__ == "__main__":
    model = DummyModel()
    # Save the model's state_dict for registry upload
    torch.save(model.state_dict(), "model.pt")
    print("Dummy PyTorch model saved as model.pt")
