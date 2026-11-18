// Placeholder data for widgets that have no backend yet (quests, balance, weekly index).
// Replace with real queries when the corresponding API modules land.
export const DEMO = {
  berries: 1450,
quests: [
    { key: "feed", target: 1, reward: 20 },
    { key: "play", target: 2, reward: 35 },
    { key: "wash", target: 1, reward: 40 },
  ],
  chestReward: 100,
  week: [62, 74, 58, 81, 90, 77, 98],
  decor: ["wallpaper", "plants", "furniture"],
  shop: [
    { id: "apple", icon: "🍎", price: 15, name: "backpack.items.apple" },
    { id: "fish", icon: "🐟", price: 30, name: "backpack.items.fish" },
    { id: "ball", icon: "🎾", price: 40, name: "backpack.items.ball" },
    { id: "soap", icon: "🧼", price: 25, name: "backpack.items.soap" },
    { id: "pastel", icon: "🎨", price: 120, name: "shop.decorItems.pastel" },
    { id: "fern", icon: "🪴", price: 80, name: "shop.decorItems.fern" },
  ],
  friends: [
    { nick: "Olesya", pet: "Bublyk", score: 1280 },
    { nick: "Taras", pet: "Pushok", score: 1120 },
    { nick: "Marko", pet: "Kulka", score: 940 },
  ],
} as const;

export const ITEM_EMOJI: Record<string, string> = { apple: "🍎", fish: "🐟", ball: "🎾", soap: "🧼", pastel: "🎨", fern: "🪴" };
