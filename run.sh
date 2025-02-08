for i in {1..10}; do
  curl -X POST http://localhost:6000/order \
       -H "Content-Type: application/json" \
       -d "{\"user\": \"User_$i\", \"item\": \"iPhone 15\", \"price\": 45000}" &
done