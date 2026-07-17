function fn() {
  var env = karate.env;
  if (!env) {
    env = 'dev';
  }
  
  var config = {
    baseUrl: 'http://lms.signal.qlabs.pro:9000'
  };
  
  karate.configure('connectTimeout', 5000);
  karate.configure('readTimeout', 5000);
  
  return config;
}
