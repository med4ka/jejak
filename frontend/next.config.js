/** @type {import('next').NextConfig} */
const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

module.exports = {
  // Avatar upload disimpan lokal di Go (POST /api/profile/avatar), URL-nya
  // berupa path relatif "/uploads/..." di DB. Rewrite ini membuat gambar
  // avatar bisa dimuat dari halaman mana pun (dashboard + halaman publik)
  // lewat origin Next sendiri, yang meneruskan ke API — tidak ada cookie
  ///sesi yang dibutuhkan untuk membaca file statis ini.
  async rewrites() {
    return [
      {
        source: "/uploads/:path*",
        destination: `${GO_API_URL}/uploads/:path*`,
      },
    ];
  },
};