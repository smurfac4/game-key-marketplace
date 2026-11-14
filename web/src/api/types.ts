export type User = {
  id: number;
  email: string;
  created_at: string;
};
export type Product = {
  id: number;
  name: string;
  slug: string;
  image_url: string;
  platform: string;
  region: string;
  genre?: string;
  description?: string;
  release_date?: string | null;
  price: string;
  currency: string;
  original_price?: string | null;
  fuzzy_match?: boolean;
};
export type Facet = {
  value: string;
  count: number;
};
export type Listing = {
  data: Product[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
    total_pages: number;
  };
  facets: {
    platform: Facet[];
    region: Facet[];
    genre: Facet[];
  };
};
export type Offer = {
  id: number;
  seller: {
    id: number;
    name: string;
    rating: number;
  };
  price: string;
  discount_price: string | null;
  effective_price: string;
  stock: number;
  currency: string;
};
export type Details = {
  data: Omit<Product, "price" | "currency"> & {
    description: string;
    offers: Offer[];
  };
  similar_products: Omit<Product, "price" | "currency">[];
};
export type CartItem = {
  offer_id: number;
  product_id: number;
  slug: string;
  name: string;
  image_url: string;
  seller: string;
  quantity: number;
  stock: number;
  unit_price: string;
  total: string;
  currency: string;
};
export type Cart = {
  items: CartItem[];
  total: string;
  currency: string;
};
export type OrderItem = {
  id: number;
  product_name: string;
  seller_name: string;
  quantity: number;
  unit_price: string;
  currency: string;
  keys: string[];
};
export type Order = {
  id: number;
  status: string;
  currency: string;
  total: string;
  created_at: string;
  items: OrderItem[];
};
export type OrderList = {
  data: Order[];
  page: number;
  total: number;
  total_pages: number;
};
export type Banner = {
  id: number;
  title: string;
  image_url: string;
  link_url: string;
};
export type Block<T> = {
  title: string;
  data: T[];
};
export type Suggestion = Pick<Product, "id" | "name" | "slug" | "image_url">;
export type Profile = {
  display_name: string;
  country_code: string;
  avatar: string;
  preferred_platform: string;
  marketing_opt_in: boolean;
  wishlist_notifications: boolean;
};
export type AccountSummary = {
  profile: Profile;
  completion: { completed: number; total: number; percent: number };
  wishlist_count: number;
  order_count: number;
  key_count: number;
  currency: string;
  wallet_balance: string;
  cashback_earned: string;
  reward_points: number;
  demo: boolean;
  platforms: string[];
  countries: string[];
};
export type Favorite = Omit<Product, "price"> & { price: string | null };
export type AuthMethods = {
  password: boolean;
  email_code: boolean;
  providers: { name: string; start_url: string }[];
};
