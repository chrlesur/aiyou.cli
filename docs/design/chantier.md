Je vais mettre à jour le README.md pour inclure les nouvelles fonctionnalités de streaming et les paramètres avancés :

```markdown
[...précédentes sections inchangées jusqu'à Caractéristiques...]

## Caractéristiques

Chat avec les assistants AI.YOU :
- Mode interactif pour des conversations continues
- Mode message unique pour des requêtes rapides
- Streaming des réponses en temps réel avec affichage progressif
- Paramètres avancés pour contrôler la génération :
  - Temperature (0.0-1.0) pour ajuster la créativité
  - Top-p (0.0-1.0) pour contrôler la diversité
  - Maximum de tokens pour limiter la longueur des réponses
- Sélection flexible des assistants
- Cache intelligent des réponses fréquentes
- Gestion des sessions et de l'authentification
- Mode debug pour un logging détaillé
- Mode silencieux pour minimiser les sorties
- Configuration flexible via fichier ou variables d'environnement

[...sections d'installation inchangées...]

## Utilisation

### Chat avec un Assistant

1. Message unique :
   ```bash
   # Utilise l'assistant par défaut
   aiyou chat "Votre message"
   
   # Assistant spécifique
   aiyou chat -a asst_xLMDUf2cWAKaU8UBFFp1LsLA "Votre message"
   ```

2. Mode interactif :
   ```bash
   # Démarre une session interactive
   aiyou chat -i
   
   # Avec un assistant spécifique
   aiyou chat -i -a asst_xLMDUf2cWAKaU8UBFFp1LsLA
   ```

3. Streaming des réponses :
   ```bash
   # Affiche la réponse en temps réel
   aiyou chat -s "Votre message"
   
   # Combine streaming et mode interactif
   aiyou chat -i -s
   ```

4. Paramètres avancés :
   ```bash
   # Ajuste la créativité (0.0-1.0)
   aiyou chat --temperature 0.7 "Votre message"
   
   # Contrôle la diversité des réponses (0.0-1.0)
   aiyou chat --top-p 0.9 "Votre message"
   
   # Limite la longueur de la réponse
   aiyou chat --max-tokens 100 "Votre message"
   
   # Combine plusieurs paramètres
   aiyou chat --temperature 0.8 --top-p 0.9 --max-tokens 150 "Votre message"
   ```

## Exemples d'Utilisation

1. Chat simple avec l'assistant par défaut :
   ```bash
   aiyou chat "Quelle est la capitale de la France ?"
   ```

2. Session interactive avec streaming :
   ```bash
   aiyou chat -i -s
   > Explique-moi la relativité
   [Assistant répond en temps réel...]
   > Peux-tu simplifier ?
   [Assistant répond...]
   > exit
   ```

3. Utilisation créative avec paramètres avancés :
   ```bash
   # Génération créative avec température élevée
   aiyou chat --temperature 0.9 "Écris un poème sur l'automne"
   
   # Réponse concise avec limite de tokens
   aiyou chat --max-tokens 50 "Résume la Révolution française"
   
   # Combinaison de paramètres pour une réponse équilibrée
   aiyou chat --temperature 0.7 --top-p 0.9 --max-tokens 200 "Génère une histoire courte"
   ```

4. Streaming avec paramètres avancés :
   ```bash
   # Réponse créative en temps réel
   aiyou chat -s --temperature 0.8 "Invente une histoire"
   ```
