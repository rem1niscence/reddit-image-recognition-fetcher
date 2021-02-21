#  -------------------------------------------------------------
#   Copyright (c) Microsoft Corporation.  All rights reserved.
#  -------------------------------------------------------------
"""
Skeleton code showing how to load and run the TensorFlow SavedModel export package from Lobe.
"""
from __future__ import absolute_import, division, print_function

import json
import os
from PIL import Image
import boto3
import numpy as np
import tflite_runtime.interpreter as tflite

s3 = boto3.resource("s3")

BASE_PATH = "/tmp/"

MODEL_FILENAME = os.getenv("MODEL_FILENAME", "classify_model.tflite")
MODEL_PATH = os.path.join(BASE_PATH, MODEL_FILENAME)

SIGNATURE_FILENAME = os.getenv("SIGNATURES_FILENAME", "signature.json")
SIGNATURE_PATH = os.path.join(BASE_PATH, SIGNATURE_FILENAME)

IMG_PATH = os.path.join(BASE_PATH, 'img')

s3.Bucket("ra-model-storage").download_file(MODEL_FILENAME, MODEL_PATH)
s3.Bucket("ra-model-storage").download_file(SIGNATURE_FILENAME, SIGNATURE_PATH)


def get_model_and_sig(model_dir):
    """Method to get name of model file. Assumes model is in the parent directory for script."""
    with open(os.path.join(model_dir, SIGNATURE_FILENAME), "r") as f:
        signature = json.load(f)
    model_file = signature.get("filename")
    if not os.path.isfile(f'{model_dir}{model_file}'):
        raise FileNotFoundError("Model file does not exist")
    return model_file, signature


def load_model(model_file):
    """Load the model from path to model file"""
    # Load TFLite model and allocate tensors.
    interpreter = tflite.Interpreter(model_path=model_file)
    interpreter.allocate_tensors()
    return interpreter


def get_prediction(image, interpreter, signature):
    """
    Predict with the TFLite interpreter!
    """
    # Combine the information about the inputs and outputs from the
    # signature.json file with the Interpreter runtime
    signature_inputs = signature.get("inputs")
    input_details = {
        detail.get("name"): detail for detail in interpreter.get_input_details()}
    model_inputs = {
        key: {**sig, **input_details.get(sig.get("name"))}
        for key, sig in signature_inputs.items()
    }
    signature_outputs = signature.get("outputs")
    output_details = {
        detail.get("name"): detail for detail in interpreter.get_output_details()}
    model_outputs = {
        key: {**sig, **output_details.get(sig.get("name"))}
        for key, sig in signature_outputs.items()
    }

    if "Image" not in model_inputs:
        raise ValueError(
            "Tensorflow Lite model doesn't have 'Image' input! Check signature.json, and please report issue to Lobe."
        )

    # process image to be compatible with the model
    input_data = process_image(image, model_inputs.get("Image").get("shape"))

    # set the input to run
    interpreter.set_tensor(model_inputs.get("Image").get("index"), input_data)
    interpreter.invoke()

    # grab our desired outputs from the interpreter!
    # un-batch since we ran an image with batch size of 1, and convert to
    # normal python types with tolist()
    outputs = {
        key: interpreter.get_tensor(value.get("index")).tolist()[0]
        for key, value in model_outputs.items()
    }
    # postprocessing! convert any byte strings to normal strings with .decode()
    for key, val in outputs.items():
        if isinstance(val, bytes):
            outputs[key] = val.decode()

    return outputs


def process_image(image, input_shape):
    """
    Given a PIL Image, center square crop and resize to fit the expected model input,
    and convert from [0,255] to [0,1] values.
    """
    width, height = image.size
    # ensure image type is compatible with model and convert if not
    if image.mode != "RGB":
        image = image.convert("RGB")
    # center crop image (you can substitute any other method to make a square
    # image, such as just resizing or padding edges with 0)
    if width != height:
        square_size = min(width, height)
        left = (width - square_size) / 2
        top = (height - square_size) / 2
        right = (width + square_size) / 2
        bottom = (height + square_size) / 2
        # Crop the center of the image
        image = image.crop((left, top, right, bottom))
    # now the image is square, resize it to be the right shape for the model
    # input
    input_width, input_height = input_shape[1:3]
    if image.width != input_width or image.height != input_height:
        image = image.resize((input_width, input_height))

    # make 0-1 float instead of 0-255 int (that PIL Image loads by default)
    image = np.asarray(image) / 255.0
    # format input as model expects
    return image.reshape(input_shape).astype(np.float32)


def main(image, model_dir):
    """
    Load the model and signature files, start the TF Lite interpreter, and run prediction on the image.

    Output prediction will be a dictionary with the same keys as the outputs in the signature.json file.
    """
    model_file, signature = get_model_and_sig(model_dir)
    interpreter = load_model(model_dir + model_file)
    prediction = get_prediction(image, interpreter, signature)
    # get list of confidences from prediction
    confidences = list(prediction.values())[0]
    # get the label name for the predicted class
    labels = signature.get("classes").get("Label")
    max_confidence = max(confidences)
    prediction["Prediction"] = labels[confidences.index(max_confidence)]
    return prediction


def lambda_handler(event, context):
    results = {}
    for record in event['Records']:
      bucket = record['s3']['bucket']['name']
      key = record['s3']['object']['key']

      print('Running Deep Learning example using Tensorflow library ...')
      print(
          'Image to be processed, from: bucket [%s], object key: [%s]' %
          (bucket, key))

      s3.Bucket(bucket).download_file(key, IMG_PATH)

      image = Image.open(IMG_PATH)
      if image.mode != "RGB":
        image = image.convert("RGB")

      prediction = main(image, BASE_PATH)

      results = prediction

    print(results)
    return results
