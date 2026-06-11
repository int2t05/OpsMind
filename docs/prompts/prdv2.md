# 初始需求
重构rag对话模块 
移除anythingllm模块 改为自己实现ragllm对话模块
基础api定义加入功能： 用户可以在前端上传多格式的文档 后端并行解析然后处理到minio 
向量数据库使用pgvector 
文档解析+ragllm 实现要仿照项目：C:\Users\int2t\Desktop\Projects\Personal\rag-engine 使用go实现
dockercompose 只要依赖pg(pgvector) minio server web 

# 约束
保证模块化 代码简洁 架构严谨
