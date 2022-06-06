from typing import Any, Tuple

import torch
from torch import Tensor
from torch.autograd import Function
from torch.nn import Linear


class MePropLinearFunction(Function):
    @staticmethod
    def jvp(ctx: Any, *grad_inputs: Any) -> Any:
        return Function.jvp(ctx, *grad_inputs)

    @staticmethod
    def forward(
        ctx: Any, input: Tensor, weight: Tensor, bias: Tensor, k: int
    ) -> Tensor:
        ctx.save_for_backward(input, weight, bias, torch.tensor(k))
        output = input.mm(weight.t())
        if bias is not None:
            output += bias.unsqueeze(0).expand_as(output)
        return output

    @staticmethod
    def backward(
        ctx: Any, grad_output: Tensor
    ) -> Tuple[Tensor, Tensor, Tensor, None]:
        input, weight, bias, k = ctx.saved_tensors
        grad_input = grad_weight = grad_bias = None
        k = int(k.detach())

        if 0 < k < weight.size(1):  # backprop with top-k selection
            device = grad_output.device
            _, indices = grad_output.abs().topk(k)
            values = grad_output.gather(-1, indices).view(-1)
            row_indices = (
                torch.arange(0, grad_output.size()[0])
                .long()
                .to(device)
                .unsqueeze_(-1)
                .repeat(1, k)
            )
            indices = torch.stack([row_indices.view(-1), indices.view(-1)])
            if grad_output.is_cuda:
                pdy = torch.cuda.sparse.FloatTensor(
                    indices, values, grad_output.size()
                )
            else:
                pdy = torch.sparse_coo_tensor(
                    indices, values, grad_output.size()
                )
            if ctx.needs_input_grad[0]:
                grad_input = torch.dsmm(pdy, weight)
            if ctx.needs_input_grad[1]:
                grad_weight = torch.dsmm(pdy.t(), input)
        if bias is not None and ctx.needs_input_grad[2]:
            grad_bias = grad_output.sum(0)

        return grad_input, grad_weight, grad_bias, None


me_prop_linear = MePropLinearFunction.apply


class MePropLinear(Linear):
    """Re-implementation of Pytorch Linear layer using the sparse
    backpropagation described in the MeProp paper."""

    _k_percentage = 0.04  # Derived from the paper
    _k: int = None

    def forward(self, input: Tensor) -> Tensor:
        return me_prop_linear(input, self.weight, self.bias, self.k)

    @classmethod
    def from_linear_layer(cls, linear_layer: Linear):
        in_features = linear_layer.in_features
        out_features = linear_layer.out_features
        device = linear_layer.weight.device
        weight = linear_layer.weight
        bias = linear_layer.bias
        new_layer = cls(
            in_features=in_features,
            out_features=out_features,
            bias=(bias is not None),
        )
        new_layer.weight = weight
        new_layer.bias = bias
        return new_layer.to(device)

    @property
    def k(self):
        if self._k is None:
            self._k = round(self.out_features * self._k_percentage)
        return self._k


def patch_backward_pass(model: torch.nn.Module):
    if isinstance(model, Linear):
        return MePropLinear.from_linear_layer(model)
    for name, layer in model.named_children():
        new_layer = patch_backward_pass(layer)
        if new_layer is not layer:
            setattr(model, name, new_layer)
    return model
