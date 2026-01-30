#!/bin/bash

echo "📦 Verificando estado del repositorio..."
git status

echo ""
read -p "✏️  Mensaje del commit: " msg

if [ -z "$msg" ]; then
  echo "❌ El mensaje no puede estar vacío"
  exit 1
fi

git add .

git commit -m "$msg"

git push

echo "✅ Cambios subidos correctamente"
