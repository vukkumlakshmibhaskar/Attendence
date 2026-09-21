@echo off
title Training Face Model on 105 Classes Pins Dataset
cd /d "%~dp0"
echo ============================================================
echo   Training Face Verifier on 105 Classes Pins Dataset
echo   Dataset: C:\Users\adseng\Downloads\105_classes_pins_dataset
echo   Output:  face_verifier_pins.json
echo ============================================================
echo.
python train_face_model.py --dataset "C:\Users\adseng\Downloads\105_classes_pins_dataset" --output "face_verifier_pins.json" --max-people 105 --max-images 20 --max-pairs 20000 --epochs 300
echo.
pause
