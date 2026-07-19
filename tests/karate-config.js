function fn() {
  var env = karate.env;
  if (!env) {
    env = 'dev';
  }
  
  var config = {
    baseUrl: 'http://127.0.0.1:8888'
  };
  
  karate.configure('connectTimeout', 5000);
  karate.configure('readTimeout', 5000);
  
  return config;
}
